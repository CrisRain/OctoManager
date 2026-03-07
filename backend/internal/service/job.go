package service

import (
	"context"
	"errors"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
)

type JobService interface {
	List(ctx context.Context) ([]dto.JobResponse, error)
	ListPaged(ctx context.Context, limit, offset int) (dto.PagedResponse[dto.JobResponse], error)
	Get(ctx context.Context, id uint64) (*dto.JobResponse, error)
	Create(ctx context.Context, req *dto.CreateJobRequest) (*dto.JobResponse, error)
	Patch(ctx context.Context, id uint64, req *dto.PatchJobRequest) (*dto.JobResponse, error)
	Cancel(ctx context.Context, id uint64) (*dto.JobResponse, error)
	Delete(ctx context.Context, id uint64) error
}

type jobService struct {
	repo            repository.JobRepository
	jobRunRepo      repository.JobRunRepository
	accountTypeRepo repository.AccountTypeRepository
	dispatcher      JobDispatcher
}

const (
	jobStatusQueued   int16 = 0
	jobStatusFailed   int16 = 3
	jobStatusCanceled int16 = 4
)

func NewJobService(
	repo repository.JobRepository,
	jobRunRepo repository.JobRunRepository,
	accountTypeRepo repository.AccountTypeRepository,
	dispatcher JobDispatcher,
) JobService {
	return &jobService{
		repo:            repo,
		jobRunRepo:      jobRunRepo,
		accountTypeRepo: accountTypeRepo,
		dispatcher:      dispatcher,
	}
}

func (s *jobService) List(ctx context.Context) ([]dto.JobResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list jobs", err)
	}
	responses := make([]dto.JobResponse, 0, len(items))
	for i := range items {
		responses = append(responses, jobToResponse(&items[i]))
	}
	return responses, nil
}

func (s *jobService) ListPaged(ctx context.Context, limit, offset int) (dto.PagedResponse[dto.JobResponse], error) {
	items, total, err := s.repo.ListPaged(ctx, limit, offset)
	if err != nil {
		return dto.PagedResponse[dto.JobResponse]{}, internalError("failed to list jobs", err)
	}
	if items == nil {
		items = []model.Job{}
	}
	responses := make([]dto.JobResponse, 0, len(items))
	for i := range items {
		responses = append(responses, jobToResponse(&items[i]))
	}
	return dto.PagedResponse[dto.JobResponse]{
		Items:  responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *jobService) Get(ctx context.Context, id uint64) (*dto.JobResponse, error) {
	if id == 0 {
		return nil, invalidInput("job id is required")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "job not found")
	}
	response := jobToResponse(item)
	if s.jobRunRepo != nil {
		if lastRun, runErr := s.jobRunRepo.GetLatestByJobID(ctx, item.ID); runErr == nil {
			response.LastRun = &dto.JobRunResponse{
				ID:           lastRun.ID,
				JobID:        lastRun.JobID,
				WorkerID:     lastRun.WorkerID,
				Attempt:      lastRun.Attempt,
				Result:       lastRun.Result,
				ErrorCode:    lastRun.ErrorCode,
				ErrorMessage: lastRun.ErrorMessage,
				StartedAt:    lastRun.StartedAt,
				EndedAt:      lastRun.EndedAt,
			}
		}
	}
	return &response, nil
}

func (s *jobService) Create(ctx context.Context, req *dto.CreateJobRequest) (*dto.JobResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	typeKey := trim(req.TypeKey)
	actionKey := trim(req.ActionKey)
	if typeKey == "" {
		return nil, invalidInput("type_key is required")
	}
	if actionKey == "" {
		return nil, invalidInput("action_key is required")
	}
	if !isJSONObject(req.Selector) {
		return nil, invalidInput("selector must be a valid JSON object")
	}
	if !isJSONObject(req.Params) {
		return nil, invalidInput("params must be a valid JSON object")
	}

	accountType, err := s.accountTypeRepo.GetByKey(ctx, typeKey)
	if err != nil {
		if isNotFound(err) {
			return nil, invalidInput("type_key does not exist")
		}
		return nil, internalError("failed to validate job type", err)
	}
	if !isGenericCategory(accountType.Category) {
		return nil, invalidInput("job type must be a generic account type")
	}

	item := &model.Job{
		TypeKey:   typeKey,
		ActionKey: actionKey,
		Selector:  normalizeJSON(req.Selector, "{}"),
		Params:    normalizeJSON(req.Params, "{}"),
		Status:    jobStatusQueued,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, internalError("failed to create job", err)
	}

	if s.dispatcher == nil {
		_, _ = s.repo.UpdateStatus(ctx, item.ID, jobStatusFailed)
		return nil, internalError("job dispatcher is not configured", errors.New("missing dispatcher"))
	}
	if err := s.dispatcher.EnqueueDispatchJob(ctx, item.ID); err != nil {
		_, _ = s.repo.UpdateStatus(ctx, item.ID, jobStatusFailed)
		return nil, internalError("failed to enqueue job", err)
	}

	response := jobToResponse(item)
	return &response, nil
}

func (s *jobService) Patch(ctx context.Context, id uint64, req *dto.PatchJobRequest) (*dto.JobResponse, error) {
	if id == 0 {
		return nil, invalidInput("job id is required")
	}
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "job not found")
	}

	hasChanges := false
	if req.TypeKey != nil {
		hasChanges = true
		trimmed := trim(*req.TypeKey)
		if trimmed == "" {
			return nil, invalidInput("type_key cannot be empty")
		}
		accountType, err := s.accountTypeRepo.GetByKey(ctx, trimmed)
		if err != nil {
			if isNotFound(err) {
				return nil, invalidInput("type_key does not exist")
			}
			return nil, internalError("failed to validate job type", err)
		}
		if !isGenericCategory(accountType.Category) {
			return nil, invalidInput("job type must be a generic account type")
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
	if req.Selector != nil {
		hasChanges = true
		if !isJSONObject(*req.Selector) {
			return nil, invalidInput("selector must be a valid JSON object")
		}
		item.Selector = normalizeJSON(*req.Selector, "{}")
	}
	if req.Params != nil {
		hasChanges = true
		if !isJSONObject(*req.Params) {
			return nil, invalidInput("params must be a valid JSON object")
		}
		item.Params = normalizeJSON(*req.Params, "{}")
	}
	if req.Status != nil {
		hasChanges = true
		if !isValidJobStatus(*req.Status) {
			return nil, invalidInput("status must be one of: 0, 1, 2, 3, 4")
		}
		item.Status = *req.Status
	}

	if !hasChanges {
		return nil, invalidInput("at least one field is required")
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, internalError("failed to update job", err)
	}
	response := jobToResponse(item)
	return &response, nil
}

func (s *jobService) Cancel(ctx context.Context, id uint64) (*dto.JobResponse, error) {
	if id == 0 {
		return nil, invalidInput("job id is required")
	}
	item, err := s.repo.UpdateStatus(ctx, id, jobStatusCanceled)
	if err != nil {
		return nil, wrapRepoError(err, "job not found")
	}
	response := jobToResponse(item)
	return &response, nil
}

func (s *jobService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return invalidInput("job id is required")
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return wrapRepoError(err, "job not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return internalError("failed to delete job", err)
	}
	return nil
}

func jobToResponse(item *model.Job) dto.JobResponse {
	if item == nil {
		return dto.JobResponse{}
	}
	return dto.JobResponse{
		ID:        item.ID,
		TypeKey:   item.TypeKey,
		ActionKey: item.ActionKey,
		Selector:  normalizeJSON(item.Selector, "{}"),
		Params:    normalizeJSON(item.Params, "{}"),
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

var _ JobService = (*jobService)(nil)
