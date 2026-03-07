package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/worker/adapter"
)

const (
	triggerModeAsync = "async"
	triggerModeSync  = "sync"
)

type TriggerService interface {
	List(ctx context.Context) ([]dto.TriggerEndpointResponse, error)
	Get(ctx context.Context, id uint64) (*dto.TriggerEndpointResponse, error)
	Create(ctx context.Context, req *dto.CreateTriggerRequest) (*dto.CreateTriggerResponse, error)
	Patch(ctx context.Context, id uint64, req *dto.PatchTriggerRequest) (*dto.TriggerEndpointResponse, error)
	Delete(ctx context.Context, id uint64) error
	Fire(ctx context.Context, slug string, rawToken string, req *dto.FireTriggerRequest) (*dto.FireTriggerResponse, error)
	// FireDirect fires a trigger without validating the per-trigger bearer token.
	// Use this when the caller has already been authenticated via an API key.
	FireDirect(ctx context.Context, slug string, req *dto.FireTriggerRequest) (*dto.FireTriggerResponse, error)
}

type triggerService struct {
	repo            repository.TriggerEndpointRepository
	accountTypeRepo repository.AccountTypeRepository
	jobRepo         repository.JobRepository
	dispatcher      JobDispatcher
	executor        JobExecutor
}

func NewTriggerService(
	repo repository.TriggerEndpointRepository,
	accountTypeRepo repository.AccountTypeRepository,
	jobRepo repository.JobRepository,
	dispatcher JobDispatcher,
	executor JobExecutor,
) TriggerService {
	return &triggerService{
		repo:            repo,
		accountTypeRepo: accountTypeRepo,
		jobRepo:         jobRepo,
		dispatcher:      dispatcher,
		executor:        executor,
	}
}

func (s *triggerService) List(ctx context.Context) ([]dto.TriggerEndpointResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list trigger endpoints", err)
	}
	responses := make([]dto.TriggerEndpointResponse, 0, len(items))
	for i := range items {
		responses = append(responses, triggerToResponse(&items[i]))
	}
	return responses, nil
}

func (s *triggerService) Get(ctx context.Context, id uint64) (*dto.TriggerEndpointResponse, error) {
	if id == 0 {
		return nil, invalidInput("trigger endpoint id is required")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "trigger endpoint not found")
	}
	response := triggerToResponse(item)
	return &response, nil
}

func (s *triggerService) Create(ctx context.Context, req *dto.CreateTriggerRequest) (*dto.CreateTriggerResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	name := trim(req.Name)
	slug := trim(req.Slug)
	typeKey := trim(req.TypeKey)
	actionKey := trim(req.ActionKey)
	mode, err := normalizeTriggerMode(req.Mode)
	if err != nil {
		return nil, err
	}

	if name == "" {
		return nil, invalidInput("name is required")
	}
	if slug == "" {
		return nil, invalidInput("slug is required")
	}
	if typeKey == "" {
		return nil, invalidInput("type_key is required")
	}
	if actionKey == "" {
		return nil, invalidInput("action_key is required")
	}
	if len(req.DefaultSelector) > 0 && !isJSONObject(req.DefaultSelector) {
		return nil, invalidInput("default_selector must be a valid JSON object")
	}
	if len(req.DefaultParams) > 0 && !isJSONObject(req.DefaultParams) {
		return nil, invalidInput("default_params must be a valid JSON object")
	}

	if _, err := s.validateGenericTypeKey(ctx, typeKey); err != nil {
		return nil, err
	}

	rawToken, err := generateSecureToken(32)
	if err != nil {
		return nil, internalError("failed to generate token", err)
	}

	item := &model.TriggerEndpoint{
		Name:            name,
		Slug:            slug,
		TypeKey:         typeKey,
		ActionKey:       actionKey,
		ExecutionMode:   mode,
		DefaultSelector: normalizeJSON(req.DefaultSelector, "{}"),
		DefaultParams:   normalizeJSON(req.DefaultParams, "{}"),
		TokenHash:       hashToken(rawToken),
		TokenPrefix:     rawToken[:8],
		Enabled:         true,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("trigger slug already exists")
		}
		return nil, internalError("failed to create trigger endpoint", err)
	}

	response := triggerToResponse(item)
	return &dto.CreateTriggerResponse{
		Endpoint: response,
		RawToken: rawToken,
	}, nil
}

func (s *triggerService) Patch(ctx context.Context, id uint64, req *dto.PatchTriggerRequest) (*dto.TriggerEndpointResponse, error) {
	if id == 0 {
		return nil, invalidInput("trigger endpoint id is required")
	}
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "trigger endpoint not found")
	}

	hasChanges := false
	if req.Name != nil {
		hasChanges = true
		trimmed := trim(*req.Name)
		if trimmed == "" {
			return nil, invalidInput("name cannot be empty")
		}
		item.Name = trimmed
	}
	if req.TypeKey != nil {
		hasChanges = true
		trimmed := trim(*req.TypeKey)
		if trimmed == "" {
			return nil, invalidInput("type_key cannot be empty")
		}
		if _, err := s.validateGenericTypeKey(ctx, trimmed); err != nil {
			return nil, err
		}
		item.TypeKey = trimmed
	}
	if req.ActionKey != nil {
		hasChanges = true
		trimmed := trim(*req.ActionKey)
		if trimmed == "" {
			return nil, invalidInput("action_key cannot be empty")
		}
		item.ActionKey = trimmed
	}
	if req.Mode != nil {
		hasChanges = true
		mode, err := normalizeTriggerMode(*req.Mode)
		if err != nil {
			return nil, err
		}
		item.ExecutionMode = mode
	}
	if req.DefaultSelector != nil {
		hasChanges = true
		if !isJSONObject(*req.DefaultSelector) {
			return nil, invalidInput("default_selector must be a valid JSON object")
		}
		item.DefaultSelector = normalizeJSON(*req.DefaultSelector, "{}")
	}
	if req.DefaultParams != nil {
		hasChanges = true
		if !isJSONObject(*req.DefaultParams) {
			return nil, invalidInput("default_params must be a valid JSON object")
		}
		item.DefaultParams = normalizeJSON(*req.DefaultParams, "{}")
	}
	if req.Enabled != nil {
		hasChanges = true
		item.Enabled = *req.Enabled
	}

	if !hasChanges {
		return nil, invalidInput("at least one field is required")
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, internalError("failed to update trigger endpoint", err)
	}

	response := triggerToResponse(item)
	return &response, nil
}

func (s *triggerService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return invalidInput("trigger endpoint id is required")
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return wrapRepoError(err, "trigger endpoint not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return internalError("failed to delete trigger endpoint", err)
	}
	return nil
}

func (s *triggerService) Fire(
	ctx context.Context,
	slug string,
	rawToken string,
	req *dto.FireTriggerRequest,
) (*dto.FireTriggerResponse, error) {
	trimmedSlug := trim(slug)
	if trimmedSlug == "" {
		return nil, invalidInput("slug is required")
	}

	endpoint, err := s.repo.GetBySlug(ctx, trimmedSlug)
	if err != nil {
		return nil, wrapRepoError(err, "trigger endpoint not found")
	}
	if !endpoint.Enabled {
		return nil, unauthorized("trigger endpoint is disabled")
	}
	if hashToken(rawToken) != endpoint.TokenHash {
		return nil, unauthorized("invalid token")
	}

	modeInput := ""
	requestSelector := json.RawMessage(nil)
	extraParams := json.RawMessage(nil)
	if req != nil {
		modeInput = req.Mode
		requestSelector = req.Selector
		extraParams = req.ExtraParams
	}

	mode, err := normalizeTriggerMode(firstNonEmpty(modeInput, endpoint.ExecutionMode))
	if err != nil {
		return nil, err
	}
	if len(requestSelector) > 0 && !isJSONObject(requestSelector) {
		return nil, invalidInput("selector must be a valid JSON object")
	}
	if len(extraParams) > 0 && !isJSONObject(extraParams) {
		return nil, invalidInput("extra_params must be a valid JSON object")
	}

	if _, err := s.validateGenericTypeKey(ctx, endpoint.TypeKey); err != nil {
		return nil, err
	}

	switch mode {
	case triggerModeAsync:
		if s.dispatcher == nil {
			return nil, internalError("job dispatcher is not configured", errors.New("missing dispatcher"))
		}
	case triggerModeSync:
		if s.executor == nil {
			return nil, internalError("job executor is not configured", errors.New("missing executor"))
		}
	}

	selector := mergeJSON(endpoint.DefaultSelector, requestSelector)
	params := mergeJSON(endpoint.DefaultParams, extraParams)
	params = injectTriggerMetadata(params, endpoint, mode, selector)

	job := &model.Job{
		TypeKey:   endpoint.TypeKey,
		ActionKey: endpoint.ActionKey,
		Selector:  selector,
		Params:    params,
		Status:    JobStatusQueued,
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, internalError("failed to create job", err)
	}

	result := &dto.FireTriggerResponse{
		Endpoint: triggerToResponse(endpoint),
		Mode:     mode,
		Queued:   mode == triggerModeAsync,
		Input: dto.TriggerExecutionInput{
			TypeKey:   job.TypeKey,
			ActionKey: job.ActionKey,
			Selector:  normalizeJSON(job.Selector, "{}"),
			Params:    normalizeJSON(job.Params, "{}"),
		},
	}

	if mode == triggerModeAsync {
		if err := s.dispatcher.EnqueueDispatchJob(ctx, job.ID); err != nil {
			_, _ = s.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed)
			return nil, internalError("job created but failed to enqueue", err)
		}
		response := jobToResponse(job)
		result.Job = &response
		return result, nil
	}

	summary, err := s.executor.ExecuteJob(ctx, job.ID)
	if err != nil {
		_, _ = s.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed)
		return nil, internalError("failed to execute trigger job", err)
	}
	if summary != nil {
		job.Status = summary.Status
		result.Output = toTriggerExecutionOutput(summary)
	}
	if current, err := s.jobRepo.GetByID(ctx, job.ID); err == nil {
		job = current
	}
	response := jobToResponse(job)
	result.Job = &response
	result.Queued = false
	return result, nil
}

func (s *triggerService) FireDirect(
	ctx context.Context,
	slug string,
	req *dto.FireTriggerRequest,
) (*dto.FireTriggerResponse, error) {
	trimmedSlug := trim(slug)
	if trimmedSlug == "" {
		return nil, invalidInput("slug is required")
	}
	endpoint, err := s.repo.GetBySlug(ctx, trimmedSlug)
	if err != nil {
		return nil, wrapRepoError(err, "trigger endpoint not found")
	}
	if !endpoint.Enabled {
		return nil, unauthorized("trigger endpoint is disabled")
	}
	// Inject a synthetic rawToken that matches the stored hash so Fire's auth
	// check passes, by calling the shared execution path directly.
	modeInput := ""
	requestSelector := json.RawMessage(nil)
	extraParams := json.RawMessage(nil)
	if req != nil {
		modeInput = req.Mode
		requestSelector = req.Selector
		extraParams = req.ExtraParams
	}
	mode, err := normalizeTriggerMode(firstNonEmpty(modeInput, endpoint.ExecutionMode))
	if err != nil {
		return nil, err
	}
	if len(requestSelector) > 0 && !isJSONObject(requestSelector) {
		return nil, invalidInput("selector must be a valid JSON object")
	}
	if len(extraParams) > 0 && !isJSONObject(extraParams) {
		return nil, invalidInput("extra_params must be a valid JSON object")
	}
	if _, err := s.validateGenericTypeKey(ctx, endpoint.TypeKey); err != nil {
		return nil, err
	}
	switch mode {
	case triggerModeAsync:
		if s.dispatcher == nil {
			return nil, internalError("job dispatcher is not configured", errors.New("missing dispatcher"))
		}
	case triggerModeSync:
		if s.executor == nil {
			return nil, internalError("job executor is not configured", errors.New("missing executor"))
		}
	}
	selector := mergeJSON(endpoint.DefaultSelector, requestSelector)
	params := mergeJSON(endpoint.DefaultParams, extraParams)
	params = injectTriggerMetadata(params, endpoint, mode, selector)
	job := &model.Job{
		TypeKey:   endpoint.TypeKey,
		ActionKey: endpoint.ActionKey,
		Selector:  selector,
		Params:    params,
		Status:    JobStatusQueued,
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, internalError("failed to create job", err)
	}
	result := &dto.FireTriggerResponse{
		Endpoint: triggerToResponse(endpoint),
		Mode:     mode,
		Queued:   mode == triggerModeAsync,
		Input: dto.TriggerExecutionInput{
			TypeKey:   job.TypeKey,
			ActionKey: job.ActionKey,
			Selector:  normalizeJSON(job.Selector, "{}"),
			Params:    normalizeJSON(job.Params, "{}"),
		},
	}
	if mode == triggerModeAsync {
		if err := s.dispatcher.EnqueueDispatchJob(ctx, job.ID); err != nil {
			_, _ = s.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed)
			return nil, internalError("job created but failed to enqueue", err)
		}
		response := jobToResponse(job)
		result.Job = &response
		return result, nil
	}
	summary, err := s.executor.ExecuteJob(ctx, job.ID)
	if err != nil {
		_, _ = s.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed)
		return nil, internalError("failed to execute trigger job", err)
	}
	if summary != nil {
		job.Status = summary.Status
		result.Output = toTriggerExecutionOutput(summary)
	}
	if current, err := s.jobRepo.GetByID(ctx, job.ID); err == nil {
		job = current
	}
	response := jobToResponse(job)
	result.Job = &response
	result.Queued = false
	return result, nil
}

func (s *triggerService) validateGenericTypeKey(ctx context.Context, typeKey string) (*model.AccountType, error) {
	accountType, err := s.accountTypeRepo.GetByKey(ctx, typeKey)
	if err != nil {
		if isNotFound(err) {
			return nil, invalidInput("type_key does not exist")
		}
		return nil, internalError("failed to validate account type", err)
	}
	if !isGenericCategory(accountType.Category) {
		return nil, invalidInput("trigger type must be a generic account type")
	}
	return accountType, nil
}

func triggerToResponse(item *model.TriggerEndpoint) dto.TriggerEndpointResponse {
	if item == nil {
		return dto.TriggerEndpointResponse{}
	}
	mode, _ := normalizeTriggerMode(item.ExecutionMode)
	return dto.TriggerEndpointResponse{
		ID:              item.ID,
		Name:            item.Name,
		Slug:            item.Slug,
		TypeKey:         item.TypeKey,
		ActionKey:       item.ActionKey,
		Mode:            mode,
		DefaultSelector: normalizeJSON(item.DefaultSelector, "{}"),
		DefaultParams:   normalizeJSON(item.DefaultParams, "{}"),
		TokenPrefix:     item.TokenPrefix,
		Enabled:         item.Enabled,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func mergeJSON(base, override json.RawMessage) json.RawMessage {
	if len(override) == 0 {
		if len(base) == 0 {
			return json.RawMessage("{}")
		}
		return normalizeJSON(base, "{}")
	}

	baseMap := decodeSpecMap(base)
	overrideMap := decodeSpecMap(override)
	for key, value := range overrideMap {
		baseMap[key] = value
	}

	merged, err := json.Marshal(baseMap)
	if err != nil {
		return normalizeJSON(base, "{}")
	}
	return merged
}

func normalizeTriggerMode(mode string) (string, error) {
	switch strings.ToLower(trim(mode)) {
	case "", triggerModeAsync:
		return triggerModeAsync, nil
	case triggerModeSync:
		return triggerModeSync, nil
	default:
		return "", invalidInput("mode must be one of: async, sync")
	}
}

func injectTriggerMetadata(
	params json.RawMessage,
	endpoint *model.TriggerEndpoint,
	mode string,
	selector json.RawMessage,
) json.RawMessage {
	value := decodeSpecMap(params)
	value["_trigger"] = map[string]any{
		"endpoint_id": endpoint.ID,
		"name":        endpoint.Name,
		"slug":        endpoint.Slug,
		"type_key":    endpoint.TypeKey,
		"action_key":  endpoint.ActionKey,
		"mode":        mode,
		"selector":    decodeSpecMap(selector),
		"fired_at":    time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return normalizeJSON(params, "{}")
	}
	return raw
}

func toTriggerExecutionOutput(summary *JobExecutionSummary) *dto.TriggerExecutionOutput {
	if summary == nil {
		return nil
	}

	results := make([]dto.TriggerExecutionAccountResult, 0, len(summary.Results))
	for _, item := range summary.Results {
		results = append(results, dto.TriggerExecutionAccountResult{
			RunID:        item.RunID,
			AccountID:    item.AccountID,
			Identifier:   item.Identifier,
			Status:       item.Status,
			Result:       item.Result,
			ErrorCode:    item.ErrorCode,
			ErrorMessage: item.ErrorMessage,
			Session:      toTriggerExecutionSession(item.Session),
			StartedAt:    item.StartedAt,
			EndedAt:      item.EndedAt,
		})
	}

	return &dto.TriggerExecutionOutput{
		JobStatus:         summary.Status,
		MatchedAccounts:   summary.MatchedAccounts,
		ProcessedAccounts: summary.ProcessedAccounts,
		ErrorCode:         summary.ErrorCode,
		ErrorMessage:      summary.ErrorMessage,
		Results:           results,
	}
}

func toTriggerExecutionSession(session *adapter.Session) *dto.TriggerExecutionSession {
	if session == nil {
		return nil
	}
	return &dto.TriggerExecutionSession{
		Type:      session.Type,
		Payload:   session.Payload,
		ExpiresAt: session.ExpiresAt,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := trim(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ TriggerService = (*triggerService)(nil)
