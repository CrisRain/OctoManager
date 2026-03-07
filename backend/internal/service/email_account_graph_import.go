package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
)

const fallbackGraphImportClientID = "dbc8e03a-b00c-46bd-ae65-b683e7707cb0"

type batchImportGraphRow struct {
	Line         int
	Address      string
	ClientID     string
	RefreshToken string
}

type batchImportGraphOAuthResult struct {
	Tenant       string
	AccessToken  string
	RefreshToken string
	TokenURL     string
	TokenExpires string
	Scope        []string
}

type BatchImportGraphPreparedResult struct {
	Total       int
	Accepted    int
	Skipped     int
	Failures    []dto.BatchImportGraphEmailFailure
	TaskRequest dto.BatchImportGraphEmailTaskRequest
}

type BatchImportGraphExecutionResult struct {
	Total    int
	Created  int
	Failed   int
	Failures []dto.BatchImportGraphEmailFailure
}

func (s *emailAccountService) BatchImportGraph(
	ctx context.Context,
	req *dto.BatchImportGraphEmailRequest,
) (*dto.BatchImportGraphEmailResponse, error) {
	prepared, err := PrepareBatchImportGraphTask(ctx, s.repo, req)
	if err != nil {
		return nil, err
	}

	response := &dto.BatchImportGraphEmailResponse{
		Total:    prepared.Total,
		Accepted: prepared.Accepted,
		Skipped:  prepared.Skipped,
		Failures: prepared.Failures,
	}
	if prepared.Accepted == 0 {
		return response, nil
	}

	job, taskID, err := createAndEnqueueAsyncJob(ctx, s.jobRepo, s.dispatcher, asyncJobSpec{
		TypeKey:   asyncJobTypeSystem,
		ActionKey: asyncJobActionBatchEmailImportGraph,
		Selector: map[string]any{
			"resource": "email_account",
			"mode":     "graph_import",
			"dedupe":   "address",
			"total":    prepared.Total,
		},
		Params: map[string]any{
			"accepted":                  prepared.Accepted,
			"skipped":                   prepared.Skipped,
			"mailbox":                   trim(req.Mailbox),
			"tenant":                    trim(req.Tenant),
			"graph_base_url":            normalizeGraphBaseURL(req.GraphBaseURL),
			"scope":                     normalizeScopeList(req.Scope),
			"default_client_id_present": trim(req.DefaultClientID) != "",
		},
	}, func(jobID uint64) (string, error) {
		return s.dispatcher.EnqueueBatchEmailImportGraph(ctx, jobID, prepared.TaskRequest)
	})
	if err != nil {
		return nil, internalError("failed to enqueue email graph batch import", err)
	}

	response.Queued = true
	response.TaskID = taskID
	response.JobID = job.ID
	return response, nil
}

func PrepareBatchImportGraphTask(
	ctx context.Context,
	repo repository.EmailAccountRepository,
	req *dto.BatchImportGraphEmailRequest,
) (BatchImportGraphPreparedResult, error) {
	if req == nil {
		return BatchImportGraphPreparedResult{}, invalidInput("payload is required")
	}
	if req.Status != 0 && req.Status != 1 {
		return BatchImportGraphPreparedResult{}, invalidInput("status must be 0 (pending) or 1 (verified)")
	}

	rows, failures, total := parseBatchImportGraphRows(req.Content, req.DefaultClientID)
	if total == 0 {
		return BatchImportGraphPreparedResult{}, invalidInput("content is required")
	}

	seen := make(map[string]struct{}, len(rows))
	acceptedRows := make([]dto.BatchImportGraphEmailTaskRow, 0, len(rows))
	for _, row := range rows {
		key := strings.ToLower(row.Address)
		if _, exists := seen[key]; exists {
			failures = append(failures, dto.BatchImportGraphEmailFailure{
				Line:    row.Line,
				Address: row.Address,
				Error:   "Duplicate email in import payload; kept the first occurrence",
			})
			continue
		}
		seen[key] = struct{}{}

		if repo != nil {
			if _, err := repo.GetByAddress(ctx, row.Address); err == nil {
				failures = append(failures, dto.BatchImportGraphEmailFailure{
					Line:    row.Line,
					Address: row.Address,
					Error:   "Email account already exists",
				})
				continue
			} else if !isNotFound(err) {
				return BatchImportGraphPreparedResult{}, internalError("failed to check existing email account", err)
			}
		}

		acceptedRows = append(acceptedRows, dto.BatchImportGraphEmailTaskRow{
			Line:         row.Line,
			Address:      row.Address,
			ClientID:     row.ClientID,
			RefreshToken: row.RefreshToken,
		})
	}

	return BatchImportGraphPreparedResult{
		Total:    total,
		Accepted: len(acceptedRows),
		Skipped:  len(failures),
		Failures: failures,
		TaskRequest: dto.BatchImportGraphEmailTaskRequest{
			Rows:         acceptedRows,
			Tenant:       trim(req.Tenant),
			Scope:        normalizeScopeList(req.Scope),
			Mailbox:      normalizeBatchImportGraphMailbox(req.Mailbox),
			GraphBaseURL: normalizeGraphBaseURL(req.GraphBaseURL),
			Status:       req.Status,
		},
	}, nil
}

func ExecuteBatchImportGraphTask(
	ctx context.Context,
	repo repository.EmailAccountRepository,
	req dto.BatchImportGraphEmailTaskRequest,
	shouldStop func() bool,
) (BatchImportGraphExecutionResult, error) {
	if repo == nil {
		return BatchImportGraphExecutionResult{}, errors.New("email repository is not configured")
	}
	if len(req.Rows) == 0 {
		return BatchImportGraphExecutionResult{}, errors.New("rows is required")
	}

	refreshScope := normalizeScopeList(req.Scope)
	if len(refreshScope) == 0 {
		refreshScope = append([]string{}, defaultGraphRefreshScopes...)
	}

	configuredTenant := trim(req.Tenant)
	mailbox := normalizeBatchImportGraphMailbox(req.Mailbox)
	graphBaseURL := normalizeGraphBaseURL(req.GraphBaseURL)

	result := BatchImportGraphExecutionResult{
		Total:    len(req.Rows),
		Created:  0,
		Failed:   0,
		Failures: []dto.BatchImportGraphEmailFailure{},
	}

	oauthCache := make(map[string]batchImportGraphOAuthResult, len(req.Rows))
	oauthErrors := make(map[string]string, len(req.Rows))

	for _, row := range req.Rows {
		if shouldStop != nil && shouldStop() {
			break
		}

		cacheKey := buildBatchImportGraphCacheKey(
			batchImportGraphRow{
				Line:         row.Line,
				Address:      row.Address,
				ClientID:     row.ClientID,
				RefreshToken: row.RefreshToken,
			},
			configuredTenant,
			refreshScope,
		)

		oauthResult, ok := oauthCache[cacheKey]
		if !ok {
			if cachedErr, exists := oauthErrors[cacheKey]; exists {
				result.Failed++
				result.Failures = append(result.Failures, dto.BatchImportGraphEmailFailure{
					Line:    row.Line,
					Address: row.Address,
					Error:   cachedErr,
				})
				continue
			}

			resolvedTenant := resolveBatchImportGraphTenant(row.Address, configuredTenant)
			refreshed, selectedTokenURL, err := refreshOutlookTokenWithFallback(ctx, refreshOutlookTokenInput{
				Tenant:       resolvedTenant,
				Username:     row.Address,
				ClientID:     row.ClientID,
				RefreshToken: row.RefreshToken,
				Scope:        refreshScope,
			})
			if err != nil {
				errMessage := err.Error()
				oauthErrors[cacheKey] = errMessage
				result.Failed++
				result.Failures = append(result.Failures, dto.BatchImportGraphEmailFailure{
					Line:    row.Line,
					Address: row.Address,
					Error:   errMessage,
				})
				continue
			}

			resolvedScope := parseScopeValues(refreshed.Scope)
			if len(resolvedScope) == 0 {
				resolvedScope = append([]string{}, refreshScope...)
			}

			tokenURL := trim(refreshed.TokenURL)
			if tokenURL == "" {
				tokenURL = trim(selectedTokenURL)
			}
			if tokenURL == "" {
				tokenURL = buildBatchImportGraphTokenURL(resolvedTenant)
			}

			refreshToken := trim(refreshed.RefreshToken)
			if refreshToken == "" {
				refreshToken = row.RefreshToken
			}

			expiresAt := ""
			if !refreshed.ExpiresAt.IsZero() {
				expiresAt = refreshed.ExpiresAt.UTC().Format(timeRFC3339)
			}

			oauthResult = batchImportGraphOAuthResult{
				Tenant:       resolvedTenant,
				AccessToken:  normalizeOAuthAccessToken(refreshed.AccessToken),
				RefreshToken: refreshToken,
				TokenURL:     tokenURL,
				TokenExpires: expiresAt,
				Scope:        resolvedScope,
			}
			oauthCache[cacheKey] = oauthResult
		}

		graphConfig, err := buildBatchImportGraphConfig(
			row.Address,
			row.ClientID,
			mailbox,
			graphBaseURL,
			oauthResult,
		)
		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, dto.BatchImportGraphEmailFailure{
				Line:    row.Line,
				Address: row.Address,
				Error:   "failed to build graph_config: " + err.Error(),
			})
			continue
		}

		if err := createEmailAccountRecord(ctx, repo, row.Address, "outlook", graphConfig, req.Status); err != nil {
			result.Failed++
			result.Failures = append(result.Failures, dto.BatchImportGraphEmailFailure{
				Line:    row.Line,
				Address: row.Address,
				Error:   err.Error(),
			})
			continue
		}

		result.Created++
	}

	return result, nil
}

func createEmailAccountRecord(
	ctx context.Context,
	repo repository.EmailAccountRepository,
	address string,
	provider string,
	graphConfig json.RawMessage,
	status int16,
) error {
	if repo == nil {
		return errors.New("email repository is not configured")
	}

	parsed, err := mail.ParseAddress(trim(address))
	if err != nil {
		return invalidInput("address must be a valid email address")
	}
	normalizedAddress := strings.ToLower(strings.TrimSpace(parsed.Address))
	if !isJSONObject(graphConfig) {
		return invalidInput("graph_config must be a valid JSON object")
	}
	if status != 0 && status != 1 {
		return invalidInput("status must be 0 (pending) or 1 (verified)")
	}

	item := &model.EmailAccount{
		Address:     normalizedAddress,
		Provider:    normalizeEmailProvider(provider, normalizedAddress),
		GraphConfig: normalizeJSON(graphConfig, "{}"),
		Status:      status,
	}
	if err := repo.Create(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return conflict("email account already exists")
		}
		return internalError("failed to create email account", err)
	}
	return nil
}

func parseBatchImportGraphRows(
	content string,
	defaultClientID string,
) ([]batchImportGraphRow, []dto.BatchImportGraphEmailFailure, int) {
	lines := strings.Split(content, "\n")
	rows := make([]batchImportGraphRow, 0, len(lines))
	failures := make([]dto.BatchImportGraphEmailFailure, 0)
	total := 0

	for i, rawLine := range lines {
		line := trim(rawLine)
		if line == "" {
			continue
		}

		total++
		row, failure := parseBatchImportGraphLine(line, i+1, defaultClientID)
		if failure != nil {
			failures = append(failures, *failure)
			continue
		}
		rows = append(rows, row)
	}

	return rows, failures, total
}

func parseBatchImportGraphLine(
	line string,
	lineNumber int,
	defaultClientID string,
) (batchImportGraphRow, *dto.BatchImportGraphEmailFailure) {
	parts := strings.Split(line, "----")
	if len(parts) < 2 {
		return batchImportGraphRow{}, &dto.BatchImportGraphEmailFailure{
			Line:    lineNumber,
			Address: line,
			Error:   "Invalid format. Supported: email----refresh_token, email----client_id----refresh_token, or email----password----client_id----refresh_token",
		}
	}

	fallbackClientID := trim(defaultClientID)
	if fallbackClientID == "" {
		fallbackClientID = fallbackGraphImportClientID
	}

	var (
		addressRaw      string
		clientIDRaw     string
		refreshTokenRaw string
	)

	switch {
	case len(parts) >= 4:
		addressRaw = parts[0]
		clientIDRaw = parts[2]
		refreshTokenRaw = parts[3]
	case len(parts) == 3:
		addressRaw = parts[0]
		clientIDRaw = parts[1]
		refreshTokenRaw = parts[2]
	default:
		addressRaw = parts[0]
		clientIDRaw = fallbackClientID
		refreshTokenRaw = parts[1]
	}

	address := strings.ToLower(trim(addressRaw))
	clientID := trim(clientIDRaw)
	refreshToken := trim(refreshTokenRaw)

	if refreshToken != "" && clientID == "" {
		clientID = fallbackClientID
	}

	if address == "" || !strings.Contains(address, "@") {
		displayAddress := address
		if displayAddress == "" {
			displayAddress = "(empty)"
		}
		return batchImportGraphRow{}, &dto.BatchImportGraphEmailFailure{
			Line:    lineNumber,
			Address: displayAddress,
			Error:   "Invalid email address",
		}
	}

	if clientID == "" || refreshToken == "" {
		return batchImportGraphRow{}, &dto.BatchImportGraphEmailFailure{
			Line:    lineNumber,
			Address: address,
			Error:   "Graph mode requires client_id and refresh_token",
		}
	}

	if _, err := mail.ParseAddress(address); err != nil {
		return batchImportGraphRow{}, &dto.BatchImportGraphEmailFailure{
			Line:    lineNumber,
			Address: address,
			Error:   "Invalid email address",
		}
	}

	return batchImportGraphRow{
		Line:         lineNumber,
		Address:      address,
		ClientID:     clientID,
		RefreshToken: refreshToken,
	}, nil
}

func resolveBatchImportGraphTenant(address string, configuredTenant string) string {
	if trimmed := trim(configuredTenant); trimmed != "" {
		return trimmed
	}
	return resolveOutlookTenant("", address)
}

func buildBatchImportGraphTokenURL(tenant string) string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(tenant))
}

func buildBatchImportGraphConfig(
	address string,
	clientID string,
	mailbox string,
	graphBaseURL string,
	oauth batchImportGraphOAuthResult,
) (json.RawMessage, error) {
	graphConfig := map[string]any{
		"auth_method":    "graph_oauth2",
		"username":       address,
		"client_id":      clientID,
		"refresh_token":  oauth.RefreshToken,
		"tenant":         oauth.Tenant,
		"token_url":      oauth.TokenURL,
		"scope":          oauth.Scope,
		"graph_base_url": normalizeGraphBaseURL(graphBaseURL),
		"mailbox":        normalizeBatchImportGraphMailbox(mailbox),
	}
	if trimmed := normalizeOAuthAccessToken(oauth.AccessToken); trimmed != "" {
		graphConfig["access_token"] = trimmed
	}
	if trimmed := trim(oauth.TokenExpires); trimmed != "" {
		graphConfig["token_expires_at"] = trimmed
	}
	return json.Marshal(graphConfig)
}

func normalizeBatchImportGraphMailbox(value string) string {
	if trimmed := trim(value); trimmed != "" {
		return trimmed
	}
	return "INBOX"
}

func buildBatchImportGraphCacheKey(
	row batchImportGraphRow,
	configuredTenant string,
	scope []string,
) string {
	scopeKey := strings.Join(normalizeScopeList(scope), " ")
	return strings.Join([]string{
		strings.ToLower(row.Address),
		row.ClientID,
		row.RefreshToken,
		trim(configuredTenant),
		scopeKey,
	}, "|")
}
