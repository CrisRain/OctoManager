package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/email/graphclient"
	"octomanger/backend/internal/email/outlookoauth"
)

var errInvalidOAuthConfig = errors.New("invalid oauth config")

const (
	defaultGraphBaseURL  = "https://graph.microsoft.com/v1.0"
	emailMailboxCacheTTL = 60 * time.Second
	emailMessageCacheTTL = 30 * time.Second
	emailDetailCacheTTL  = 90 * time.Second
)

var defaultGraphRefreshScopes = []string{
	"https://graph.microsoft.com/.default",
}

func (s *emailAccountService) ListMessages(
	ctx context.Context,
	accountID uint64,
	query *dto.ListEmailMessagesQuery,
) (*dto.ListEmailMessagesResponse, error) {
	if accountID == 0 {
		return nil, invalidInput("email account id is required")
	}

	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, wrapRepoError(err, "email account not found")
	}

	updatedConfig, refreshErr := s.refreshAndSaveGraphToken(ctx, accountID, account.GraphConfig)
	runtimeCfg, err := parseGraphRuntimeConfig(updatedConfig)
	if err != nil {
		return nil, invalidInput(formatGraphOAuthConfigError(err, refreshErr))
	}

	mailbox := ""
	limit := 20
	offset := 0
	if query != nil {
		mailbox = trim(query.Mailbox)
		if query.Limit > 0 {
			limit = query.Limit
		}
		offset = query.Offset
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	targetMailbox := mailbox
	if targetMailbox == "" {
		targetMailbox = runtimeCfg.Mailbox
	}
	cacheKey := buildEmailCacheKey(
		"messages:list",
		fmt.Sprintf("%d", accountID),
		strings.ToLower(strings.TrimSpace(targetMailbox)),
		fmt.Sprintf("%d", limit),
		fmt.Sprintf("%d", offset),
	)
	var cached dto.ListEmailMessagesResponse
	if s.getCachedJSON(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	folderID, folderName, err := graphclient.ResolveMailFolder(ctx, runtimeCfg.Client, targetMailbox)
	if err != nil {
		return nil, invalidInput("failed to resolve mailbox: " + err.Error())
	}

	items, total, err := graphclient.ListMessages(ctx, runtimeCfg.Client, folderID, limit, offset)
	if err != nil {
		return nil, invalidInput("failed to list email messages: " + err.Error())
	}

	resultItems := make([]dto.EmailMessageSummary, 0, len(items))
	for _, item := range items {
		resultItems = append(resultItems, dto.EmailMessageSummary{
			ID:      item.ID,
			Subject: item.Subject,
			From:    item.From,
			To:      item.To,
			Date:    item.Date,
			Size:    item.Size,
			Flags:   item.Flags,
		})
	}

	result := &dto.ListEmailMessagesResponse{
		Mailbox: folderName,
		Limit:   limit,
		Offset:  offset,
		Total:   total,
		Items:   resultItems,
	}
	s.setCachedJSON(ctx, cacheKey, result, emailMessageCacheTTL)
	return result, nil
}

func (s *emailAccountService) GetMessage(
	ctx context.Context,
	accountID uint64,
	_ string,
	messageID string,
) (*dto.EmailMessageDetail, error) {
	if accountID == 0 {
		return nil, invalidInput("email account id is required")
	}
	if trim(messageID) == "" {
		return nil, invalidInput("message id is required")
	}

	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, wrapRepoError(err, "email account not found")
	}

	updatedConfig, refreshErr := s.refreshAndSaveGraphToken(ctx, accountID, account.GraphConfig)
	runtimeCfg, err := parseGraphRuntimeConfig(updatedConfig)
	if err != nil {
		return nil, invalidInput(formatGraphOAuthConfigError(err, refreshErr))
	}
	cacheKey := buildEmailCacheKey("messages:detail", fmt.Sprintf("%d", accountID), trim(messageID))
	var cached dto.EmailMessageDetail
	if s.getCachedJSON(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	item, err := graphclient.GetMessage(ctx, runtimeCfg.Client, messageID)
	if err != nil {
		return nil, invalidInput("failed to get email message: " + err.Error())
	}

	result := &dto.EmailMessageDetail{
		ID:       item.ID,
		Subject:  item.Subject,
		From:     item.From,
		To:       item.To,
		Cc:       item.Cc,
		Date:     item.Date,
		Size:     item.Size,
		Flags:    item.Flags,
		Headers:  item.Headers,
		TextBody: item.TextBody,
		HTMLBody: item.HTMLBody,
	}
	s.setCachedJSON(ctx, cacheKey, result, emailDetailCacheTTL)
	return result, nil
}

func (s *emailAccountService) ListMailboxes(
	ctx context.Context,
	accountID uint64,
	query *dto.ListEmailMailboxesQuery,
) (*dto.ListEmailMailboxesResponse, error) {
	if accountID == 0 {
		return nil, invalidInput("email account id is required")
	}

	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, wrapRepoError(err, "email account not found")
	}

	updatedConfig, refreshErr := s.refreshAndSaveGraphToken(ctx, accountID, account.GraphConfig)
	runtimeCfg, err := parseGraphRuntimeConfig(updatedConfig)
	if err != nil {
		return nil, invalidInput(formatGraphOAuthConfigError(err, refreshErr))
	}

	pattern := ""
	reference := ""
	if query != nil {
		pattern = trim(query.Pattern)
		reference = trim(query.Reference)
	}
	cacheKey := buildEmailCacheKey(
		"mailboxes:list",
		fmt.Sprintf("%d", accountID),
		strings.ToLower(pattern),
		strings.ToLower(reference),
	)
	var cached dto.ListEmailMailboxesResponse
	if s.getCachedJSON(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	items, err := graphclient.ListMailFolders(ctx, runtimeCfg.Client, 200)
	if err != nil {
		return nil, invalidInput("failed to list mailboxes: " + err.Error())
	}

	filtered := make([]dto.EmailMailbox, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.DisplayName)
		if !matchMailboxPattern(name, pattern) {
			continue
		}
		filtered = append(filtered, dto.EmailMailbox{
			Name:  name,
			Flags: []string{},
		})
	}

	result := &dto.ListEmailMailboxesResponse{
		Reference: reference,
		Pattern:   pattern,
		Items:     filtered,
	}
	s.setCachedJSON(ctx, cacheKey, result, emailMailboxCacheTTL)
	return result, nil
}

func (s *emailAccountService) GetLatestMessage(
	ctx context.Context,
	accountID uint64,
	query *dto.ListEmailMessagesQuery,
) (*dto.LatestEmailMessageResponse, error) {
	mailbox := ""
	if query != nil {
		mailbox = query.Mailbox
	}

	listResult, err := s.ListMessages(ctx, accountID, &dto.ListEmailMessagesQuery{
		Mailbox: mailbox,
		Limit:   1,
		Offset:  0,
	})
	if err != nil {
		return nil, err
	}
	if len(listResult.Items) == 0 {
		return &dto.LatestEmailMessageResponse{
			Mailbox: listResult.Mailbox,
			Found:   false,
		}, nil
	}

	detail, err := s.GetMessage(ctx, accountID, listResult.Mailbox, listResult.Items[0].ID)
	if err != nil {
		return nil, err
	}

	return &dto.LatestEmailMessageResponse{
		Mailbox: listResult.Mailbox,
		Found:   true,
		Item:    detail,
	}, nil
}

func (s *emailAccountService) PreviewLatestMessage(
	ctx context.Context,
	req *dto.PreviewEmailRequest,
) (*dto.LatestEmailMessageResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	if !isJSONObject(req.GraphConfig) {
		return nil, invalidInput("graph_config must be a valid JSON object")
	}

	updatedConfig, refreshErr := s.refreshAndSaveGraphToken(ctx, 0, req.GraphConfig)
	runtimeCfg, err := parseGraphRuntimeConfig(updatedConfig)
	if err != nil {
		return nil, invalidInput(formatGraphOAuthConfigError(err, refreshErr))
	}

	targetMailbox := trim(req.Mailbox)
	if targetMailbox == "" {
		targetMailbox = runtimeCfg.Mailbox
	}

	folderID, folderName, err := graphclient.ResolveMailFolder(ctx, runtimeCfg.Client, targetMailbox)
	if err != nil {
		return nil, invalidInput("failed to resolve mailbox: " + err.Error())
	}

	items, _, err := graphclient.ListMessages(ctx, runtimeCfg.Client, folderID, 1, 0)
	if err != nil {
		return nil, invalidInput("failed to get latest email message: " + err.Error())
	}
	if len(items) == 0 {
		return &dto.LatestEmailMessageResponse{
			Mailbox: folderName,
			Found:   false,
		}, nil
	}

	detailRaw, err := graphclient.GetMessage(ctx, runtimeCfg.Client, items[0].ID)
	if err != nil {
		return nil, invalidInput("failed to get latest email message: " + err.Error())
	}

	return &dto.LatestEmailMessageResponse{
		Mailbox: folderName,
		Found:   true,
		Item: &dto.EmailMessageDetail{
			ID:       detailRaw.ID,
			Subject:  detailRaw.Subject,
			From:     detailRaw.From,
			To:       detailRaw.To,
			Cc:       detailRaw.Cc,
			Date:     detailRaw.Date,
			Size:     detailRaw.Size,
			Flags:    detailRaw.Flags,
			Headers:  detailRaw.Headers,
			TextBody: detailRaw.TextBody,
			HTMLBody: detailRaw.HTMLBody,
		},
	}, nil
}

func (s *emailAccountService) PreviewMailboxes(
	ctx context.Context,
	req *dto.PreviewEmailRequest,
) (*dto.ListEmailMailboxesResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	if !isJSONObject(req.GraphConfig) {
		return nil, invalidInput("graph_config must be a valid JSON object")
	}

	updatedConfig, refreshErr := s.refreshAndSaveGraphToken(ctx, 0, req.GraphConfig)
	runtimeCfg, err := parseGraphRuntimeConfig(updatedConfig)
	if err != nil {
		return nil, invalidInput(formatGraphOAuthConfigError(err, refreshErr))
	}

	items, err := graphclient.ListMailFolders(ctx, runtimeCfg.Client, 200)
	if err != nil {
		return nil, invalidInput("failed to list mailboxes: " + err.Error())
	}

	pattern := trim(req.Pattern)
	resultItems := make([]dto.EmailMailbox, 0, len(items))
	for _, item := range items {
		if !matchMailboxPattern(item.DisplayName, pattern) {
			continue
		}
		resultItems = append(resultItems, dto.EmailMailbox{
			Name:  strings.TrimSpace(item.DisplayName),
			Flags: []string{},
		})
	}

	return &dto.ListEmailMailboxesResponse{
		Reference: trim(req.Reference),
		Pattern:   pattern,
		Items:     resultItems,
	}, nil
}

type graphRuntimeConfig struct {
	Client  graphclient.Config
	Mailbox string
}

type graphConfigPayload struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	SSL                *bool  `json:"ssl,omitempty"`
	StartTLS           bool   `json:"starttls,omitempty"`
	Username           string `json:"username,omitempty"`
	AuthMethod         string `json:"auth_method,omitempty"`
	AccessToken        string `json:"access_token,omitempty"`
	RefreshToken       string `json:"refresh_token,omitempty"`
	ClientID           string `json:"client_id,omitempty"`
	ClientSecret       string `json:"client_secret,omitempty"`
	Tenant             string `json:"tenant,omitempty"`
	Scope              any    `json:"scope,omitempty"`
	TokenURL           string `json:"token_url,omitempty"`
	TokenExpiresAt     string `json:"token_expires_at,omitempty"`
	Mailbox            string `json:"mailbox,omitempty"`
	TimeoutSeconds     int    `json:"timeout_seconds,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	GraphBaseURL       string `json:"graph_base_url,omitempty"`
}

func parseGraphRuntimeConfig(raw json.RawMessage) (graphRuntimeConfig, error) {
	if len(raw) == 0 {
		return graphRuntimeConfig{}, fmt.Errorf("%w: graph_config is empty", errInvalidOAuthConfig)
	}

	var payload graphConfigPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return graphRuntimeConfig{}, fmt.Errorf("%w: graph_config is not valid JSON", errInvalidOAuthConfig)
	}

	accessToken := normalizeOAuthAccessToken(payload.AccessToken)
	if accessToken == "" {
		return graphRuntimeConfig{}, fmt.Errorf("%w: access_token is required", errInvalidOAuthConfig)
	}

	timeout := 20 * time.Second
	if payload.TimeoutSeconds > 0 {
		timeout = time.Duration(payload.TimeoutSeconds) * time.Second
	}

	mailbox := trim(payload.Mailbox)

	return graphRuntimeConfig{
		Client: graphclient.Config{
			AccessToken: accessToken,
			BaseURL:     normalizeGraphBaseURL(payload.GraphBaseURL),
			Timeout:     timeout,
		},
		Mailbox: mailbox,
	}, nil
}

func formatGraphOAuthConfigError(parseErr error, refreshErr error) string {
	if parseErr == nil {
		return "invalid oauth config"
	}
	message := "invalid oauth config: " + parseErr.Error()
	if refreshErr != nil {
		message += "; refresh failed: " + refreshErr.Error()
	}
	return message
}

func shouldRefreshOutlookToken(accessToken string, refreshToken string, expiresAt time.Time) bool {
	if trim(refreshToken) == "" {
		return false
	}
	accessToken = normalizeOAuthAccessToken(accessToken)
	if accessToken == "" {
		return true
	}
	if expiresAt.IsZero() {
		expiresAt = extractOAuthTokenExpiry(accessToken)
	}
	return isOutlookTokenExpiring(expiresAt)
}

func isOutlookTokenExpiring(expiresAt time.Time) bool {
	if expiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().Add(tokenRefreshGrace).After(expiresAt.UTC())
}

func parseTokenExpiry(value string) time.Time {
	candidate := trim(value)
	if candidate == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, candidate)
	if err == nil {
		return parsed
	}
	return time.Time{}
}

func resolveOutlookTenant(rawTenant string, username string) string {
	if trimmed := trim(rawTenant); trimmed != "" {
		return trimmed
	}
	if isOutlookConsumerAddress(username) {
		return "consumers"
	}
	return "common"
}

func isOutlookConsumerAddress(username string) bool {
	lowerUser := strings.ToLower(strings.TrimSpace(username))
	at := strings.LastIndex(lowerUser, "@")
	if at < 0 || at == len(lowerUser)-1 {
		return false
	}
	switch lowerUser[at+1:] {
	case "outlook.com", "hotmail.com", "live.com", "msn.com":
		return true
	default:
		return false
	}
}

func resolveOAuthUsername(username string, accessToken string) string {
	if fromToken := extractOAuthUsername(accessToken); fromToken != "" {
		return fromToken
	}
	return trim(username)
}

func extractOAuthUsername(accessToken string) string {
	parts := strings.Split(normalizeOAuthAccessToken(accessToken), ".")
	if len(parts) < 2 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	for _, key := range []string{"preferred_username", "upn", "email", "unique_name"} {
		if value, ok := claims[key].(string); ok && trim(value) != "" {
			return trim(value)
		}
	}
	return ""
}

func extractOAuthTokenExpiry(accessToken string) time.Time {
	parts := strings.Split(normalizeOAuthAccessToken(accessToken), ".")
	if len(parts) < 2 {
		return time.Time{}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}
	}

	raw, ok := claims["exp"]
	if !ok {
		return time.Time{}
	}
	unixSeconds, ok := parseUnixTimestampClaim(raw)
	if !ok || unixSeconds <= 0 {
		return time.Time{}
	}
	// Some providers send ms precision in numeric claims.
	if unixSeconds >= 1_000_000_000_000 {
		unixSeconds /= 1000
	}

	return time.Unix(unixSeconds, 0).UTC()
}

func parseUnixTimestampClaim(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int32:
		return int64(typed), true
	case int:
		return int64(typed), true
	case uint64:
		if typed > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return parsed, true
		}
		floatParsed, floatErr := typed.Float64()
		if floatErr == nil {
			return int64(floatParsed), true
		}
		return 0, false
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return parsed, true
		}
		if floatParsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return int64(floatParsed), true
		}
		return 0, false
	default:
		return 0, false
	}
}

func parseScopeValues(value any) []string {
	switch typed := value.(type) {
	case nil:
		return []string{}
	case string:
		split := strings.Fields(strings.ReplaceAll(typed, ",", " "))
		return normalizeScopeList(split)
	case []any:
		items := make([]string, 0, len(typed))
		for _, raw := range typed {
			asString, ok := raw.(string)
			if !ok {
				continue
			}
			items = append(items, asString)
		}
		return normalizeScopeList(items)
	case []string:
		return normalizeScopeList(typed)
	default:
		return []string{}
	}
}

func normalizeOAuthAccessToken(value string) string {
	token := strings.TrimSpace(value)
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	return token
}

type refreshOutlookTokenInput struct {
	Tenant       string
	Username     string
	ClientID     string
	ClientSecret string
	RefreshToken string
	Scope        []string
	TokenURL     string
}

func shouldRetryRefreshWithoutClientSecret(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "aadsts7000215") {
		return true
	}
	return strings.Contains(message, "invalid_client") && strings.Contains(message, "client secret")
}

func refreshOutlookTokenOnce(
	ctx context.Context,
	tenant string,
	clientID string,
	clientSecret string,
	refreshToken string,
	scope []string,
	tokenURL string,
) (outlookoauth.TokenResponse, error) {
	return outlookoauth.RefreshToken(ctx, outlookoauth.RefreshTokenInput{
		Tenant:       tenant,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		Scope:        scope,
		TokenURL:     tokenURL,
	})
}

func refreshOutlookTokenWithFallback(
	ctx context.Context,
	input refreshOutlookTokenInput,
) (outlookoauth.TokenResponse, string, error) {
	tenant := trim(input.Tenant)
	if tenant == "" {
		tenant = "common"
	}
	tenantCandidates := uniqueNonEmptyStrings(
		[]string{
			tenant,
			resolveOutlookTenant("", input.Username),
			"common",
		},
	)
	if isOutlookConsumerAddress(input.Username) {
		tenantCandidates = uniqueNonEmptyStrings(append(tenantCandidates, "consumers"))
	}

	scopeCandidates := buildGraphRefreshScopeCandidates(input.Scope)

	var (
		lastErr           error
		firstEndpointErr  error
		firstMalformedErr error
		lastTokenURLTried string
	)
	for _, tenantCandidate := range tenantCandidates {
		defaultTokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantCandidate)
		tokenURLCandidates := uniqueNonEmptyStrings([]string{trim(input.TokenURL), defaultTokenURL})
		if len(tokenURLCandidates) == 0 {
			tokenURLCandidates = []string{defaultTokenURL}
		}

		for _, tokenURLCandidate := range tokenURLCandidates {
			for _, scopeCandidate := range scopeCandidates {
				lastTokenURLTried = tokenURLCandidate
				clientID := trim(input.ClientID)
				clientSecret := trim(input.ClientSecret)
				refreshToken := trim(input.RefreshToken)

				refreshed, err := refreshOutlookTokenOnce(
					ctx,
					tenantCandidate,
					clientID,
					clientSecret,
					refreshToken,
					scopeCandidate,
					tokenURLCandidate,
				)
				if err != nil && clientSecret != "" && shouldRetryRefreshWithoutClientSecret(err) {
					refreshed, err = refreshOutlookTokenOnce(
						ctx,
						tenantCandidate,
						clientID,
						"",
						refreshToken,
						scopeCandidate,
						tokenURLCandidate,
					)
				}
				if err != nil {
					lastErr = err
					if firstEndpointErr == nil {
						firstEndpointErr = err
					}
					continue
				}

				normalizedToken := normalizeOAuthAccessToken(refreshed.AccessToken)
				if normalizedToken == "" {
					lastErr = fmt.Errorf(
						"token endpoint returned empty access_token (token_url=%s tenant=%s scope=%s)",
						tokenURLCandidate,
						tenantCandidate,
						scopeCandidateLabel(scopeCandidate),
					)
					if firstMalformedErr == nil {
						firstMalformedErr = lastErr
					}
					continue
				}

				refreshed.AccessToken = normalizedToken
				return refreshed, tokenURLCandidate, nil
			}
		}
	}

	if firstEndpointErr != nil {
		if firstMalformedErr != nil {
			return outlookoauth.TokenResponse{}, lastTokenURLTried, fmt.Errorf(
				"%w; also received invalid access_token from token endpoint (%v)",
				firstEndpointErr,
				firstMalformedErr,
			)
		}
		return outlookoauth.TokenResponse{}, lastTokenURLTried, firstEndpointErr
	}
	if firstMalformedErr != nil {
		return outlookoauth.TokenResponse{}, lastTokenURLTried, fmt.Errorf(
			"token refresh returned invalid access_token: %v",
			firstMalformedErr,
		)
	}
	if lastErr == nil {
		lastErr = errors.New("token refresh failed")
	}
	return outlookoauth.TokenResponse{}, lastTokenURLTried, lastErr
}

func uniqueNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := trim(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func buildGraphRefreshScopeCandidates(input []string) [][]string {
	_ = input
	return [][]string{append([]string{}, defaultGraphRefreshScopes...)}
}

func containsGraphScope(scopes []string) bool {
	for _, scope := range scopes {
		lower := strings.ToLower(trim(scope))
		if strings.Contains(lower, "graph.microsoft.com/") {
			return true
		}
	}
	return false
}

func uniqueScopeCandidates(candidates [][]string) [][]string {
	if len(candidates) == 0 {
		return [][]string{nil}
	}
	result := make([][]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		normalized := uniqueNonEmptyStrings(candidate)
		key := scopeCandidateLabel(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if len(normalized) == 0 {
			result = append(result, nil)
			continue
		}
		result = append(result, normalized)
	}
	if len(result) == 0 {
		return [][]string{nil}
	}
	return result
}

func scopeCandidateLabel(scopes []string) string {
	if len(scopes) == 0 {
		return "<default>"
	}
	return strings.Join(scopes, " ")
}

// refreshAndSaveGraphToken refreshes OAuth tokens for Graph-based message access and persists them.
func (s *emailAccountService) refreshAndSaveGraphToken(
	ctx context.Context,
	accountID uint64,
	graphConfig json.RawMessage,
) (json.RawMessage, error) {
	var payload graphConfigPayload
	if err := json.Unmarshal(graphConfig, &payload); err != nil {
		return graphConfig, nil
	}

	accessToken := normalizeOAuthAccessToken(payload.AccessToken)
	refreshToken := trim(payload.RefreshToken)
	tokenExpiresAt := parseTokenExpiry(payload.TokenExpiresAt)

	if !shouldRefreshOutlookToken(accessToken, refreshToken, tokenExpiresAt) {
		return graphConfig, nil
	}

	clientID := trim(payload.ClientID)
	if clientID == "" {
		return graphConfig, errors.New("client_id is required for token refresh")
	}

	username := trim(payload.Username)
	tenant := resolveOutlookTenant(payload.Tenant, username)
	tokenURL := trim(payload.TokenURL)
	scope := parseScopeValues(payload.Scope)
	clientSecret := trim(payload.ClientSecret)

	refreshed, selectedTokenURL, err := refreshOutlookTokenWithFallback(ctx, refreshOutlookTokenInput{
		Tenant:       tenant,
		Username:     username,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		Scope:        scope,
		TokenURL:     tokenURL,
	})
	if err != nil {
		return graphConfig, err
	}
	refreshedAccessToken := normalizeOAuthAccessToken(refreshed.AccessToken)
	if refreshedAccessToken == "" {
		return graphConfig, fmt.Errorf("token endpoint returned empty access_token (token_url=%s)", selectedTokenURL)
	}

	expiresAt := ""
	if !refreshed.ExpiresAt.IsZero() {
		expiresAt = refreshed.ExpiresAt.UTC().Format(time.RFC3339)
	}

	newScope := parseScopeValues(refreshed.Scope)
	if len(newScope) == 0 {
		newScope = scope
	}
	newTokenURL := trim(refreshed.TokenURL)
	if newTokenURL == "" {
		newTokenURL = selectedTokenURL
	}

	updated := patchGraphConfigToken(
		graphConfig,
		refreshedAccessToken,
		expiresAt,
		trim(refreshed.RefreshToken),
		newTokenURL,
		newScope,
	)

	if accountID > 0 && s.repo != nil {
		if account, getErr := s.repo.GetByID(ctx, accountID); getErr == nil {
			account.GraphConfig = updated
			_ = s.repo.Update(ctx, account)
		}
	}

	return updated, nil
}

func normalizeGraphBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultGraphBaseURL
	}
	return strings.TrimSuffix(trimmed, "/")
}

func matchMailboxPattern(name string, pattern string) bool {
	target := strings.ToLower(strings.TrimSpace(name))
	rawPattern := strings.TrimSpace(pattern)
	if rawPattern == "" || rawPattern == "*" {
		return true
	}

	p := strings.ToLower(rawPattern)
	p = strings.ReplaceAll(p, "%", "*")
	if strings.HasPrefix(p, "*") && strings.HasSuffix(p, "*") && len(p) >= 2 {
		return strings.Contains(target, strings.Trim(p, "*"))
	}
	if strings.HasPrefix(p, "*") {
		return strings.HasSuffix(target, strings.TrimPrefix(p, "*"))
	}
	if strings.HasSuffix(p, "*") {
		return strings.HasPrefix(target, strings.TrimSuffix(p, "*"))
	}
	return target == p
}
