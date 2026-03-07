package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/worker/bridge"
)

type EmailAccountService interface {
	List(ctx context.Context) ([]dto.EmailAccountResponse, error)
	ListPaged(ctx context.Context, limit, offset int) (dto.PagedResponse[dto.EmailAccountResponse], error)
	Get(ctx context.Context, id uint64) (*dto.EmailAccountResponse, error)
	Create(ctx context.Context, req *dto.CreateEmailAccountRequest) (*dto.EmailAccountResponse, error)
	Patch(ctx context.Context, id uint64, req *dto.PatchEmailAccountRequest) (*dto.EmailAccountResponse, error)
	Delete(ctx context.Context, id uint64) error
	Verify(ctx context.Context, id uint64) (*dto.EmailAccountResponse, error)

	BatchDelete(ctx context.Context, req *dto.BatchEmailAccountIDsRequest) (dto.BatchResult, error)
	BatchVerify(ctx context.Context, req *dto.BatchEmailAccountIDsRequest) (dto.BatchResult, error)
	BatchImportGraph(ctx context.Context, req *dto.BatchImportGraphEmailRequest) (*dto.BatchImportGraphEmailResponse, error)
	BatchRegister(ctx context.Context, req *dto.BatchRegisterEmailRequest) (*dto.BatchRegisterEmailResponse, error)

	ListMessages(ctx context.Context, accountID uint64, query *dto.ListEmailMessagesQuery) (*dto.ListEmailMessagesResponse, error)
	GetMessage(ctx context.Context, accountID uint64, mailbox string, messageID string) (*dto.EmailMessageDetail, error)
	ListMailboxes(ctx context.Context, accountID uint64, query *dto.ListEmailMailboxesQuery) (*dto.ListEmailMailboxesResponse, error)
	GetLatestMessage(ctx context.Context, accountID uint64, query *dto.ListEmailMessagesQuery) (*dto.LatestEmailMessageResponse, error)
	PreviewLatestMessage(ctx context.Context, req *dto.PreviewEmailRequest) (*dto.LatestEmailMessageResponse, error)
	PreviewMailboxes(ctx context.Context, req *dto.PreviewEmailRequest) (*dto.ListEmailMailboxesResponse, error)

	BuildOutlookAuthorizeURL(ctx context.Context, req *dto.OutlookAuthorizeURLRequest) (*dto.OutlookAuthorizeURLResponse, error)
	ExchangeOutlookCode(ctx context.Context, req *dto.OutlookExchangeCodeRequest) (*dto.OutlookTokenResponse, error)
	RefreshOutlookToken(ctx context.Context, req *dto.OutlookRefreshTokenRequest) (*dto.OutlookTokenResponse, error)
}

type emailAccountService struct {
	repo           repository.EmailAccountRepository
	batchRegistrar EmailBatchRegistrar
	dispatcher     JobDispatcher
	cacheClient    *redis.Client
	jobRepo        repository.JobRepository
}

func NewEmailAccountService(
	repo repository.EmailAccountRepository,
	registrar EmailBatchRegistrar,
	dispatcher JobDispatcher,
	cacheClient *redis.Client,
	jobRepo repository.JobRepository,
) EmailAccountService {
	return &emailAccountService{
		repo:           repo,
		batchRegistrar: registrar,
		dispatcher:     dispatcher,
		cacheClient:    cacheClient,
		jobRepo:        jobRepo,
	}
}

func (s *emailAccountService) List(ctx context.Context) ([]dto.EmailAccountResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list email accounts", err)
	}
	responses := make([]dto.EmailAccountResponse, 0, len(items))
	for i := range items {
		responses = append(responses, emailAccountToResponse(&items[i]))
	}
	return responses, nil
}

func (s *emailAccountService) ListPaged(ctx context.Context, limit, offset int) (dto.PagedResponse[dto.EmailAccountResponse], error) {
	items, total, err := s.repo.ListPaged(ctx, limit, offset)
	if err != nil {
		return dto.PagedResponse[dto.EmailAccountResponse]{}, internalError("failed to list email accounts", err)
	}
	if items == nil {
		items = []model.EmailAccount{}
	}

	responses := make([]dto.EmailAccountResponse, 0, len(items))
	for i := range items {
		responses = append(responses, emailAccountToResponse(&items[i]))
	}
	return dto.PagedResponse[dto.EmailAccountResponse]{
		Items:  responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *emailAccountService) Get(ctx context.Context, id uint64) (*dto.EmailAccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("email account id is required")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "email account not found")
	}
	response := emailAccountToResponse(item)
	return &response, nil
}

func (s *emailAccountService) Create(ctx context.Context, req *dto.CreateEmailAccountRequest) (*dto.EmailAccountResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	address := trim(req.Address)
	provider := trim(req.Provider)
	if address == "" {
		return nil, invalidInput("address is required")
	}
	parsed, err := mail.ParseAddress(address)
	if err != nil {
		return nil, invalidInput("address must be a valid email address")
	}
	address = strings.ToLower(strings.TrimSpace(parsed.Address))
	graphConfig := req.GraphConfig
	if !isJSONObject(graphConfig) {
		return nil, invalidInput("graph_config must be a valid JSON object")
	}
	if req.Status != 0 && req.Status != 1 {
		return nil, invalidInput("status must be 0 (pending) or 1 (verified)")
	}

	normalizedProvider := normalizeEmailProvider(provider, address)

	item := &model.EmailAccount{
		Address:     address,
		Provider:    normalizedProvider,
		GraphConfig: normalizeJSON(graphConfig, "{}"),
		Status:      req.Status,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("email account already exists")
		}
		return nil, internalError("failed to create email account", err)
	}

	response := emailAccountToResponse(item)
	return &response, nil
}

func (s *emailAccountService) Patch(ctx context.Context, id uint64, req *dto.PatchEmailAccountRequest) (*dto.EmailAccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("email account id is required")
	}
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "email account not found")
	}

	hasChanges := false
	address := item.Address
	if req.Address != nil {
		hasChanges = true
		trimmed := trim(*req.Address)
		if trimmed == "" {
			return nil, invalidInput("address cannot be empty")
		}
		parsed, parseErr := mail.ParseAddress(trimmed)
		if parseErr != nil {
			return nil, invalidInput("address must be a valid email address")
		}
		address = strings.ToLower(strings.TrimSpace(parsed.Address))
		item.Address = address
	}

	shouldNormalizeProvider := false
	provider := item.Provider
	if req.Provider != nil {
		hasChanges = true
		shouldNormalizeProvider = true
		provider = trim(*req.Provider)
	} else if req.Address != nil {
		shouldNormalizeProvider = true
		provider = ""
	}
	if shouldNormalizeProvider {
		item.Provider = normalizeEmailProvider(provider, address)
	}

	if graphConfig := req.GraphConfig; graphConfig != nil {
		hasChanges = true
		if !isJSONObject(*graphConfig) {
			return nil, invalidInput("graph_config must be a valid JSON object")
		}
		item.GraphConfig = normalizeJSON(*graphConfig, "{}")
	}
	if req.Status != nil {
		hasChanges = true
		if *req.Status != 0 && *req.Status != 1 {
			return nil, invalidInput("status must be 0 (pending) or 1 (verified)")
		}
		item.Status = *req.Status
	}

	if !hasChanges {
		return nil, invalidInput("at least one field is required")
	}

	if err := s.repo.Update(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("email account already exists")
		}
		return nil, internalError("failed to update email account", err)
	}

	response := emailAccountToResponse(item)
	return &response, nil
}

func (s *emailAccountService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return invalidInput("email account id is required")
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return wrapRepoError(err, "email account not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return internalError("failed to delete email account", err)
	}
	return nil
}

func (s *emailAccountService) Verify(ctx context.Context, id uint64) (*dto.EmailAccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("email account id is required")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "email account not found")
	}
	item.Status = 1
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, internalError("failed to verify email account", err)
	}
	response := emailAccountToResponse(item)
	return &response, nil
}

func (s *emailAccountService) BatchDelete(ctx context.Context, req *dto.BatchEmailAccountIDsRequest) (dto.BatchResult, error) {
	if req == nil || len(req.IDs) == 0 {
		return dto.BatchResult{}, invalidInput("ids is required")
	}
	job, taskID, err := createAndEnqueueAsyncJob(ctx, s.jobRepo, s.dispatcher, asyncJobSpec{
		TypeKey:   asyncJobTypeSystem,
		ActionKey: asyncJobActionBatchEmailDelete,
		Selector: map[string]any{
			"resource": "email_account",
			"total":    len(req.IDs),
		},
		Params: map[string]any{
			"ids_count": len(req.IDs),
		},
	}, func(jobID uint64) (string, error) {
		return s.dispatcher.EnqueueBatchEmailDelete(ctx, jobID, *req)
	})
	if err != nil {
		return dto.BatchResult{}, internalError("failed to enqueue email batch delete", err)
	}
	return dto.BatchResult{
		Total:    len(req.IDs),
		Success:  0,
		Failed:   0,
		Failures: []dto.BatchFailure{},
		Queued:   true,
		TaskID:   taskID,
		JobID:    job.ID,
	}, nil
}

func (s *emailAccountService) BatchVerify(ctx context.Context, req *dto.BatchEmailAccountIDsRequest) (dto.BatchResult, error) {
	if req == nil || len(req.IDs) == 0 {
		return dto.BatchResult{}, invalidInput("ids is required")
	}
	job, taskID, err := createAndEnqueueAsyncJob(ctx, s.jobRepo, s.dispatcher, asyncJobSpec{
		TypeKey:   asyncJobTypeSystem,
		ActionKey: asyncJobActionBatchEmailVerify,
		Selector: map[string]any{
			"resource": "email_account",
			"total":    len(req.IDs),
		},
		Params: map[string]any{
			"ids_count": len(req.IDs),
		},
	}, func(jobID uint64) (string, error) {
		return s.dispatcher.EnqueueBatchEmailVerify(ctx, jobID, *req)
	})
	if err != nil {
		return dto.BatchResult{}, internalError("failed to enqueue email batch verify", err)
	}
	return dto.BatchResult{
		Total:    len(req.IDs),
		Success:  0,
		Failed:   0,
		Failures: []dto.BatchFailure{},
		Queued:   true,
		TaskID:   taskID,
		JobID:    job.ID,
	}, nil
}

func (s *emailAccountService) BatchRegister(ctx context.Context, req *dto.BatchRegisterEmailRequest) (*dto.BatchRegisterEmailResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	if req.Count <= 0 {
		return nil, invalidInput("count must be > 0")
	}
	if req.Count > 200 {
		return nil, invalidInput("count must be <= 200")
	}
	if req.Status != 0 && req.Status != 1 {
		return nil, invalidInput("status must be 0 (pending) or 1 (verified)")
	}
	job, taskID, err := createAndEnqueueAsyncJob(ctx, s.jobRepo, s.dispatcher, asyncJobSpec{
		TypeKey:   asyncJobTypeSystem,
		ActionKey: asyncJobActionBatchEmailRegister,
		Selector: map[string]any{
			"resource": "email_account",
			"total":    req.Count,
		},
		Params: map[string]any{
			"provider":               trim(req.Provider),
			"count":                  req.Count,
			"prefix":                 trim(req.Prefix),
			"domain":                 trim(req.Domain),
			"start_index":            req.StartIndex,
			"status":                 req.Status,
			"graph_defaults_present": len(req.GraphDefaults) > 0,
			"graph_defaults_key_count": func() int {
				if len(req.GraphDefaults) == 0 {
					return 0
				}
				var value map[string]any
				if err := json.Unmarshal(req.GraphDefaults, &value); err != nil {
					return 0
				}
				return len(value)
			}(),
		},
	}, func(jobID uint64) (string, error) {
		return s.dispatcher.EnqueueBatchEmailRegister(ctx, jobID, *req)
	})
	if err != nil {
		return nil, internalError("failed to enqueue email batch register", err)
	}
	return &dto.BatchRegisterEmailResponse{
		Requested: req.Count,
		Generated: 0,
		Created:   0,
		Failed:    0,
		Provider:  req.Provider,
		Accounts:  []dto.EmailAccountResponse{},
		Failures:  []dto.BatchRegisterEmailFailure{},
		Queued:    true,
		TaskID:    taskID,
		JobID:     job.ID,
	}, nil
}

func emailAccountToResponse(item *model.EmailAccount) dto.EmailAccountResponse {
	if item == nil {
		return dto.EmailAccountResponse{}
	}
	return dto.EmailAccountResponse{
		ID:           item.ID,
		Address:      item.Address,
		Provider:     item.Provider,
		Status:       item.Status,
		GraphSummary: buildEmailConfigSummary(item.GraphConfig),
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func buildEmailConfigSummary(raw json.RawMessage) *dto.EmailConfigSummary {
	if len(raw) == 0 {
		return nil
	}

	var payload graphConfigPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	username := trim(payload.Username)
	authMethod := strings.ToLower(trim(payload.AuthMethod))
	if authMethod == "" {
		authMethod = "auto"
	}
	accessToken := trim(payload.AccessToken)
	mailbox := trim(payload.Mailbox)

	useSSL := true
	if payload.SSL != nil {
		useSSL = *payload.SSL
	} else if payload.Port == 143 || payload.Port == 587 {
		useSSL = false
	}

	tokenExpiresAt := trim(payload.TokenExpiresAt)
	tenant := trim(payload.Tenant)
	if authMethod == "xoauth2" || trim(payload.ClientID) != "" || trim(payload.RefreshToken) != "" {
		tenant = resolveOutlookTenant(payload.Tenant, username)
	}

	return &dto.EmailConfigSummary{
		Host:                trim(payload.Host),
		Port:                payload.Port,
		SSL:                 useSSL,
		StartTLS:            payload.StartTLS,
		Username:            username,
		TokenUsername:       extractOAuthUsername(accessToken),
		AuthMethod:          authMethod,
		Tenant:              tenant,
		Mailbox:             mailbox,
		Scope:               parseScopeValues(payload.Scope),
		TokenExpiresAt:      tokenExpiresAt,
		AccessTokenPresent:  accessToken != "",
		RefreshTokenPresent: trim(payload.RefreshToken) != "",
		ClientIDPresent:     trim(payload.ClientID) != "",
		ClientSecretPresent: trim(payload.ClientSecret) != "",
	}
}

func normalizeEmailProvider(provider string, address string) string {
	value := strings.ToLower(trim(provider))
	if value != "" {
		return value
	}

	addressParts := strings.Split(strings.ToLower(address), "@")
	if len(addressParts) != 2 {
		return "custom"
	}

	domain := addressParts[1]
	switch {
	case strings.Contains(domain, "gmail.com"):
		return "gmail"
	case strings.Contains(domain, "outlook.com"), strings.Contains(domain, "hotmail.com"), strings.Contains(domain, "live.com"):
		return "outlook"
	case strings.Contains(domain, "qq.com"):
		return "qq"
	case strings.Contains(domain, "163.com"):
		return "163"
	default:
		return "custom"
	}
}

type BatchRegisterCandidate struct {
	Address     string
	Provider    string
	Status      int16
	GraphConfig json.RawMessage
}

type EmailBatchPreparedResult struct {
	Requested  int
	Provider   string
	Candidates []BatchRegisterCandidate
}

type EmailBatchRegistrar interface {
	Prepare(ctx context.Context, input dto.BatchRegisterEmailRequest) (EmailBatchPreparedResult, error)
}

type EmailPythonRunner interface {
	Execute(ctx context.Context, input bridge.Input) (bridge.Output, error)
}

type PythonEmailBatchRegistrar struct {
	runner EmailPythonRunner
}

func NewPythonEmailBatchRegistrar(runner EmailPythonRunner) *PythonEmailBatchRegistrar {
	return &PythonEmailBatchRegistrar{runner: runner}
}

func (r *PythonEmailBatchRegistrar) Prepare(
	ctx context.Context,
	input dto.BatchRegisterEmailRequest,
) (EmailBatchPreparedResult, error) {
	if r == nil || r.runner == nil {
		return EmailBatchPreparedResult{}, errors.New("python runner is not configured")
	}

	operator := strings.TrimSpace(input.Operator)
	if operator == "" {
		operator = "batch-operator"
	}

	params := map[string]any{
		"provider":    strings.TrimSpace(input.Provider),
		"count":       input.Count,
		"prefix":      strings.TrimSpace(input.Prefix),
		"domain":      strings.TrimSpace(input.Domain),
		"start_index": input.StartIndex,
		"status":      input.Status,
	}
	if len(input.Options) > 0 {
		params["options"] = input.Options
	}

	requestID := fmt.Sprintf("batch-email-register:%s", operator)
	if token, tokenErr := generateSecureToken(6); tokenErr == nil {
		requestID = fmt.Sprintf("batch-email-register:%s:%s", operator, token)
	}

	output, err := r.runner.Execute(ctx, bridge.Input{
		Action: "BATCH_REGISTER_EMAIL",
		Account: bridge.InputAccount{
			Identifier: operator,
			Spec: map[string]any{
				"provider": strings.TrimSpace(input.Provider),
			},
		},
		Params: params,
		Context: bridge.InputContext{
			RequestID: requestID,
		},
	})
	if err != nil {
		return EmailBatchPreparedResult{}, err
	}

	if !strings.EqualFold(output.Status, "success") {
		message := strings.TrimSpace(output.ErrorMessage)
		if message == "" {
			message = "python batch register returned error"
		}
		code := strings.TrimSpace(output.ErrorCode)
		if code != "" {
			return EmailBatchPreparedResult{}, fmt.Errorf("%s: %s", code, message)
		}
		return EmailBatchPreparedResult{}, errors.New(message)
	}

	candidates, provider, requested, err := parseBatchRegisterCandidates(output.Result)
	if err != nil {
		return EmailBatchPreparedResult{}, err
	}

	return EmailBatchPreparedResult{
		Requested:  requested,
		Provider:   provider,
		Candidates: candidates,
	}, nil
}

func parseBatchRegisterCandidates(
	result map[string]any,
) ([]BatchRegisterCandidate, string, int, error) {
	if result == nil {
		return nil, "", 0, errors.New("python result is empty")
	}

	rawProvider := strings.TrimSpace(toString(result["provider"]))
	requested := toInt(result["requested"])

	rawCandidates, ok := result["generated"].([]any)
	if !ok {
		if rawAccounts, ok := result["accounts"].([]any); ok {
			rawCandidates = rawAccounts
		}
	}
	if len(rawCandidates) == 0 {
		return []BatchRegisterCandidate{}, rawProvider, requested, nil
	}

	candidates := make([]BatchRegisterCandidate, 0, len(rawCandidates))
	for i, entry := range rawCandidates {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, "", 0, fmt.Errorf("invalid generated[%d]: must be object", i)
		}

		address := strings.TrimSpace(toString(item["address"]))
		if address == "" {
			return nil, "", 0, fmt.Errorf("invalid generated[%d]: address is required", i)
		}
		provider := strings.TrimSpace(toString(item["provider"]))

		rawGraphConfig, exists := item["graph_config"]
		if !exists {
			return nil, "", 0, fmt.Errorf("invalid generated[%d].graph_config: graph_config is required", i)
		}
		graphConfigRaw, err := toRawJSONObject(rawGraphConfig, "{}")
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid generated[%d].graph_config: %w", i, err)
		}
		status := int16(0)
		if rawStatus, exists := item["status"]; exists {
			parsed := toInt(rawStatus)
			status = int16(parsed)
		}

		candidates = append(candidates, BatchRegisterCandidate{
			Address:     address,
			Provider:    provider,
			Status:      status,
			GraphConfig: graphConfigRaw,
		})
	}
	return candidates, rawProvider, requested, nil
}

func toRawJSONObject(value any, fallback string) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage(fallback), nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var test map[string]any
	if err := json.Unmarshal(raw, &test); err != nil {
		return nil, errors.New("must be a JSON object")
	}
	return raw, nil
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func toInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return 0
}

var _ EmailAccountService = (*emailAccountService)(nil)
