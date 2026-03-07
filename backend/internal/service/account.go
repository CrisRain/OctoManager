package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/email/outlookoauth"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
)

type AccountService interface {
	List(ctx context.Context) ([]dto.AccountResponse, error)
	ListPaged(ctx context.Context, limit, offset int, typeKey string) (dto.PagedResponse[dto.AccountResponse], error)
	Get(ctx context.Context, id uint64) (*dto.AccountResponse, error)
	Create(ctx context.Context, req *dto.CreateAccountRequest) (*dto.AccountResponse, error)
	Patch(ctx context.Context, id uint64, req *dto.PatchAccountRequest) (*dto.AccountResponse, error)
	Delete(ctx context.Context, id uint64) error
	BatchPatch(ctx context.Context, req *dto.BatchPatchAccountRequest) (dto.BatchResult, error)
	BatchDelete(ctx context.Context, req *dto.BatchDeleteAccountRequest) (dto.BatchResult, error)
}

type accountService struct {
	repo            repository.AccountRepository
	accountTypeRepo repository.AccountTypeRepository
	dispatcher      JobDispatcher
	jobRepo         repository.JobRepository
}

func NewAccountService(
	repo repository.AccountRepository,
	accountTypeRepo repository.AccountTypeRepository,
	dispatcher JobDispatcher,
	jobRepo repository.JobRepository,
) AccountService {
	return &accountService{
		repo:            repo,
		accountTypeRepo: accountTypeRepo,
		dispatcher:      dispatcher,
		jobRepo:         jobRepo,
	}
}

func (s *accountService) List(ctx context.Context) ([]dto.AccountResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list accounts", err)
	}
	responses := make([]dto.AccountResponse, 0, len(items))
	for i := range items {
		responses = append(responses, accountToResponse(&items[i]))
	}
	return responses, nil
}

func (s *accountService) ListPaged(ctx context.Context, limit, offset int, typeKey string) (dto.PagedResponse[dto.AccountResponse], error) {
	items, total, err := s.repo.ListPaged(ctx, limit, offset, typeKey)
	if err != nil {
		return dto.PagedResponse[dto.AccountResponse]{}, internalError("failed to list accounts", err)
	}

	if items == nil {
		items = []model.Account{}
	}

	for i := range items {
		items[i] = tryRefreshAccountOAuth(ctx, items[i], func(id uint64, spec json.RawMessage) (*model.Account, error) {
			item, err := s.repo.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
			item.Spec = spec
			if err := s.repo.Update(ctx, item); err != nil {
				return nil, err
			}
			return item, nil
		})
	}

	responses := make([]dto.AccountResponse, 0, len(items))
	for i := range items {
		responses = append(responses, accountToResponse(&items[i]))
	}
	return dto.PagedResponse[dto.AccountResponse]{
		Items:  responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *accountService) Get(ctx context.Context, id uint64) (*dto.AccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("account id is required")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "account not found")
	}
	response := accountToResponse(item)
	return &response, nil
}

func (s *accountService) Create(ctx context.Context, req *dto.CreateAccountRequest) (*dto.AccountResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	typeKey := trim(req.TypeKey)
	identifier := trim(req.Identifier)
	if typeKey == "" {
		return nil, invalidInput("type_key is required")
	}
	if identifier == "" {
		return nil, invalidInput("identifier is required")
	}
	if !isJSONObject(req.Spec) {
		return nil, invalidInput("spec must be a valid JSON object")
	}

	accountType, err := s.accountTypeRepo.GetByKey(ctx, typeKey)
	if err != nil {
		if isNotFound(err) {
			return nil, invalidInput("type_key does not exist")
		}
		return nil, internalError("failed to validate account type", err)
	}
	if !isGenericCategory(accountType.Category) {
		return nil, invalidInput("accounts only support generic account types")
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	item := &model.Account{
		TypeKey:    typeKey,
		Identifier: identifier,
		Status:     req.Status,
		Tags:       model.NewStringArray(tags),
		Spec:       normalizeJSON(req.Spec, "{}"),
	}

	if err := s.repo.Create(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("account identifier already exists for type")
		}
		return nil, internalError("failed to create account", err)
	}

	response := accountToResponse(item)
	return &response, nil
}

func (s *accountService) Patch(ctx context.Context, id uint64, req *dto.PatchAccountRequest) (*dto.AccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("account id is required")
	}
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "account not found")
	}

	hasChanges := false
	if req.Status != nil {
		hasChanges = true
		item.Status = *req.Status
	}
	if req.Tags != nil {
		hasChanges = true
		item.Tags = model.NewStringArray(req.Tags)
	}
	if req.Spec != nil {
		hasChanges = true
		if !isJSONObject(*req.Spec) {
			return nil, invalidInput("spec must be a valid JSON object")
		}
		item.Spec = normalizeJSON(*req.Spec, "{}")
	}
	if !hasChanges {
		return nil, invalidInput("at least one field is required")
	}

	if err := s.repo.Update(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("account identifier already exists for type")
		}
		return nil, internalError("failed to update account", err)
	}

	response := accountToResponse(item)
	return &response, nil
}

func (s *accountService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return invalidInput("account id is required")
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return wrapRepoError(err, "account not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return internalError("failed to delete account", err)
	}
	return nil
}

func (s *accountService) BatchPatch(ctx context.Context, req *dto.BatchPatchAccountRequest) (dto.BatchResult, error) {
	if req == nil || len(req.IDs) == 0 {
		return dto.BatchResult{}, invalidInput("ids is required")
	}
	if req.Status == nil && req.Tags == nil {
		return dto.BatchResult{}, invalidInput("status or tags is required")
	}
	job, taskID, err := createAndEnqueueAsyncJob(ctx, s.jobRepo, s.dispatcher, asyncJobSpec{
		TypeKey:   asyncJobTypeSystem,
		ActionKey: asyncJobActionBatchAccountPatch,
		Selector: map[string]any{
			"resource": "account",
			"total":    len(req.IDs),
		},
		Params: map[string]any{
			"ids_count": len(req.IDs),
			"status":    req.Status,
			"tags_count": func() int {
				if req.Tags == nil {
					return 0
				}
				return len(req.Tags)
			}(),
		},
	}, func(jobID uint64) (string, error) {
		return s.dispatcher.EnqueueBatchAccountPatch(ctx, jobID, *req)
	})
	if err != nil {
		return dto.BatchResult{}, internalError("failed to enqueue account batch patch", err)
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

func (s *accountService) BatchDelete(ctx context.Context, req *dto.BatchDeleteAccountRequest) (dto.BatchResult, error) {
	if req == nil || len(req.IDs) == 0 {
		return dto.BatchResult{}, invalidInput("ids is required")
	}
	job, taskID, err := createAndEnqueueAsyncJob(ctx, s.jobRepo, s.dispatcher, asyncJobSpec{
		TypeKey:   asyncJobTypeSystem,
		ActionKey: asyncJobActionBatchAccountDelete,
		Selector: map[string]any{
			"resource": "account",
			"total":    len(req.IDs),
		},
		Params: map[string]any{
			"ids_count": len(req.IDs),
		},
	}, func(jobID uint64) (string, error) {
		return s.dispatcher.EnqueueBatchAccountDelete(ctx, jobID, *req)
	})
	if err != nil {
		return dto.BatchResult{}, internalError("failed to enqueue account batch delete", err)
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

func accountToResponse(item *model.Account) dto.AccountResponse {
	if item == nil {
		return dto.AccountResponse{}
	}
	return dto.AccountResponse{
		ID:         item.ID,
		TypeKey:    item.TypeKey,
		Identifier: item.Identifier,
		Status:     item.Status,
		Tags:       item.Tags.Slice(),
		Spec:       normalizeJSON(item.Spec, "{}"),
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

const tokenRefreshGrace = 5 * time.Minute

func tryRefreshAccountOAuth(
	ctx context.Context,
	account model.Account,
	patchFn func(id uint64, spec json.RawMessage) (*model.Account, error),
) model.Account {
	spec := decodeSpecMap(account.Spec)

	refreshToken := strings.TrimSpace(asString(spec["refresh_token"]))
	clientID := strings.TrimSpace(asString(spec["client_id"]))
	clientSecret := strings.TrimSpace(asString(spec["client_secret"]))
	if refreshToken == "" || clientID == "" {
		return account
	}

	accessToken := normalizeOAuthAccessToken(asString(spec["access_token"]))
	needsRefresh := accessToken == ""
	if !needsRefresh {
		expiresAt := time.Time{}
		if expiresAtStr := strings.TrimSpace(asString(spec["token_expires_at"])); expiresAtStr != "" {
			if parsed, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				expiresAt = parsed
			}
		}
		if expiresAt.IsZero() {
			expiresAt = extractOAuthTokenExpiry(accessToken)
		}
		needsRefresh = isOutlookTokenExpiring(expiresAt)
	}
	if !needsRefresh {
		return account
	}

	resp, err := outlookoauth.RefreshToken(ctx, outlookoauth.RefreshTokenInput{
		Tenant:       normalizeOutlookTenant(asString(spec["tenant"])),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		Scope:        specScopeList(spec),
		TokenURL:     strings.TrimSpace(asString(spec["token_url"])),
	})
	if err != nil {
		return account
	}

	spec["access_token"] = normalizeOAuthAccessToken(resp.AccessToken)
	spec["token_expires_at"] = resp.ExpiresAt.UTC().Format(time.RFC3339)
	if resp.RefreshToken != "" {
		spec["refresh_token"] = resp.RefreshToken
	}

	newSpec, err := json.Marshal(spec)
	if err != nil {
		return account
	}
	updated, err := patchFn(account.ID, newSpec)
	if err != nil {
		return account
	}
	return *updated
}

func decodeSpecMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]any{}
	}
	if value == nil {
		return map[string]any{}
	}
	return value
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func specScopeList(spec map[string]any) []string {
	switch v := spec["scope"].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		return strings.Fields(trimmed)
	}
	return nil
}

var _ AccountService = (*accountService)(nil)
