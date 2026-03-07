package taskhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/service"
	"octomanger/backend/internal/task"
)

type BatchHandlerOptions struct {
	Logger           *zap.Logger
	AccountRepo      repository.AccountRepository
	EmailAccountRepo repository.EmailAccountRepository
	JobRepo          repository.JobRepository
	JobRunRepo       repository.JobRunRepository
	BatchRegistrar   service.EmailBatchRegistrar
	WorkerID         string
}

type BatchHandler struct {
	logger         *zap.Logger
	accountRepo    repository.AccountRepository
	emailRepo      repository.EmailAccountRepository
	jobRepo        repository.JobRepository
	jobRunRepo     repository.JobRunRepository
	batchRegistrar service.EmailBatchRegistrar
	workerID       string
}

func NewBatchHandler(opts BatchHandlerOptions) *BatchHandler {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	workerID := strings.TrimSpace(opts.WorkerID)
	if workerID == "" {
		host, err := os.Hostname()
		if err != nil {
			host = "worker"
		}
		workerID = fmt.Sprintf("%s:%d", host, os.Getpid())
	}

	return &BatchHandler{
		logger:         logger,
		accountRepo:    opts.AccountRepo,
		emailRepo:      opts.EmailAccountRepo,
		jobRepo:        opts.JobRepo,
		jobRunRepo:     opts.JobRunRepo,
		batchRegistrar: opts.BatchRegistrar,
		workerID:       workerID,
	}
}

func (h *BatchHandler) ProcessAccountBatchPatch(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseBatchAccountPatchPayload(t)
	if err != nil {
		h.logger.Warn("invalid batch account patch payload", zap.Error(err))
		return asynq.SkipRetry
	}
	if !h.beginTrackedJob(ctx, payload.JobID) {
		return nil
	}

	req := payload.Request
	success := 0
	failed := 0
	canceled := false

	for _, id := range req.IDs {
		if h.isTrackedJobCanceled(ctx, payload.JobID) {
			canceled = true
			break
		}
		if id == 0 {
			failed++
			continue
		}
		item, getErr := h.accountRepo.GetByID(ctx, id)
		if getErr != nil {
			if !errors.Is(getErr, gorm.ErrRecordNotFound) {
				h.logger.Warn("batch account patch: failed to load account", zap.Uint64("account_id", id), zap.Error(getErr))
			}
			failed++
			continue
		}
		if req.Status != nil {
			item.Status = *req.Status
		}
		if req.Tags != nil {
			item.Tags = model.NewStringArray(req.Tags)
		}
		if updateErr := h.accountRepo.Update(ctx, item); updateErr != nil {
			h.logger.Warn("batch account patch: failed to update account", zap.Uint64("account_id", id), zap.Error(updateErr))
			failed++
			continue
		}
		success++
	}

	h.completeTrackedJob(ctx, payload.JobID, map[string]any{
		"total":    len(req.IDs),
		"success":  success,
		"failed":   failed,
		"canceled": canceled,
	}, batchSummaryStatus(canceled, failed))

	h.logger.Info("batch account patch finished",
		zap.Int("total", len(req.IDs)),
		zap.Int("success", success),
		zap.Int("failed", failed),
		zap.Bool("canceled", canceled),
	)
	return nil
}

func (h *BatchHandler) ProcessAccountBatchDelete(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseBatchAccountDeletePayload(t)
	if err != nil {
		h.logger.Warn("invalid batch account delete payload", zap.Error(err))
		return asynq.SkipRetry
	}
	if !h.beginTrackedJob(ctx, payload.JobID) {
		return nil
	}

	req := payload.Request
	success := 0
	failed := 0
	canceled := false

	for _, id := range req.IDs {
		if h.isTrackedJobCanceled(ctx, payload.JobID) {
			canceled = true
			break
		}
		if id == 0 {
			failed++
			continue
		}
		if _, getErr := h.accountRepo.GetByID(ctx, id); getErr != nil {
			if !errors.Is(getErr, gorm.ErrRecordNotFound) {
				h.logger.Warn("batch account delete: failed to load account", zap.Uint64("account_id", id), zap.Error(getErr))
			}
			failed++
			continue
		}
		if deleteErr := h.accountRepo.Delete(ctx, id); deleteErr != nil {
			h.logger.Warn("batch account delete: failed to delete account", zap.Uint64("account_id", id), zap.Error(deleteErr))
			failed++
			continue
		}
		success++
	}

	h.completeTrackedJob(ctx, payload.JobID, map[string]any{
		"total":    len(req.IDs),
		"success":  success,
		"failed":   failed,
		"canceled": canceled,
	}, batchSummaryStatus(canceled, failed))

	h.logger.Info("batch account delete finished",
		zap.Int("total", len(req.IDs)),
		zap.Int("success", success),
		zap.Int("failed", failed),
		zap.Bool("canceled", canceled),
	)
	return nil
}

func (h *BatchHandler) ProcessEmailBatchDelete(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseBatchEmailDeletePayload(t)
	if err != nil {
		h.logger.Warn("invalid batch email delete payload", zap.Error(err))
		return asynq.SkipRetry
	}
	if !h.beginTrackedJob(ctx, payload.JobID) {
		return nil
	}

	req := payload.Request
	success := 0
	failed := 0
	canceled := false

	for _, id := range req.IDs {
		if h.isTrackedJobCanceled(ctx, payload.JobID) {
			canceled = true
			break
		}
		if id == 0 {
			failed++
			continue
		}
		if _, getErr := h.emailRepo.GetByID(ctx, id); getErr != nil {
			if !errors.Is(getErr, gorm.ErrRecordNotFound) {
				h.logger.Warn("batch email delete: failed to load account", zap.Uint64("account_id", id), zap.Error(getErr))
			}
			failed++
			continue
		}
		if deleteErr := h.emailRepo.Delete(ctx, id); deleteErr != nil {
			h.logger.Warn("batch email delete: failed to delete account", zap.Uint64("account_id", id), zap.Error(deleteErr))
			failed++
			continue
		}
		success++
	}

	h.completeTrackedJob(ctx, payload.JobID, map[string]any{
		"total":    len(req.IDs),
		"success":  success,
		"failed":   failed,
		"canceled": canceled,
	}, batchSummaryStatus(canceled, failed))

	h.logger.Info("batch email delete finished",
		zap.Int("total", len(req.IDs)),
		zap.Int("success", success),
		zap.Int("failed", failed),
		zap.Bool("canceled", canceled),
	)
	return nil
}

func (h *BatchHandler) ProcessEmailBatchVerify(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseBatchEmailVerifyPayload(t)
	if err != nil {
		h.logger.Warn("invalid batch email verify payload", zap.Error(err))
		return asynq.SkipRetry
	}
	if !h.beginTrackedJob(ctx, payload.JobID) {
		return nil
	}

	req := payload.Request
	success := 0
	failed := 0
	canceled := false

	for _, id := range req.IDs {
		if h.isTrackedJobCanceled(ctx, payload.JobID) {
			canceled = true
			break
		}
		if id == 0 {
			failed++
			continue
		}
		item, getErr := h.emailRepo.GetByID(ctx, id)
		if getErr != nil {
			if !errors.Is(getErr, gorm.ErrRecordNotFound) {
				h.logger.Warn("batch email verify: failed to load account", zap.Uint64("account_id", id), zap.Error(getErr))
			}
			failed++
			continue
		}
		item.Status = 1
		if updateErr := h.emailRepo.Update(ctx, item); updateErr != nil {
			h.logger.Warn("batch email verify: failed to update account", zap.Uint64("account_id", id), zap.Error(updateErr))
			failed++
			continue
		}
		success++
	}

	h.completeTrackedJob(ctx, payload.JobID, map[string]any{
		"total":    len(req.IDs),
		"success":  success,
		"failed":   failed,
		"canceled": canceled,
	}, batchSummaryStatus(canceled, failed))

	h.logger.Info("batch email verify finished",
		zap.Int("total", len(req.IDs)),
		zap.Int("success", success),
		zap.Int("failed", failed),
		zap.Bool("canceled", canceled),
	)
	return nil
}

func (h *BatchHandler) ProcessEmailBatchRegister(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseBatchEmailRegisterPayload(t)
	if err != nil {
		h.logger.Warn("invalid batch email register payload", zap.Error(err))
		return asynq.SkipRetry
	}
	if !h.beginTrackedJob(ctx, payload.JobID) {
		return nil
	}

	req := payload.Request
	if req.Count <= 0 || req.Count > 200 {
		h.completeTrackedJob(ctx, payload.JobID, map[string]any{
			"requested": req.Count,
			"error":     "count must be between 1 and 200",
		}, jobStatusFailed)
		h.logger.Warn("batch email register rejected: invalid count", zap.Int("count", req.Count))
		return asynq.SkipRetry
	}
	if req.Status != 0 && req.Status != 1 {
		h.completeTrackedJob(ctx, payload.JobID, map[string]any{
			"requested": req.Count,
			"error":     "status must be 0 or 1",
		}, jobStatusFailed)
		h.logger.Warn("batch email register rejected: invalid status", zap.Int16("status", req.Status))
		return asynq.SkipRetry
	}
	if h.batchRegistrar == nil {
		h.completeTrackedJob(ctx, payload.JobID, map[string]any{
			"requested": req.Count,
			"error":     "batch registrar is not configured",
		}, jobStatusFailed)
		return asynq.SkipRetry
	}

	prepared, err := h.batchRegistrar.Prepare(ctx, req)
	if err != nil {
		h.completeTrackedJob(ctx, payload.JobID, map[string]any{
			"requested": req.Count,
			"error":     err.Error(),
		}, jobStatusFailed)
		h.logger.Error("batch email register prepare failed", zap.Error(err))
		return nil
	}

	created := 0
	failed := 0
	canceled := false
	for _, candidate := range prepared.Candidates {
		if h.isTrackedJobCanceled(ctx, payload.JobID) {
			canceled = true
			break
		}

		status := candidate.Status
		if status != 0 && status != 1 {
			status = req.Status
		}

		graphConfig := candidate.GraphConfig
		if len(req.GraphDefaults) > 0 {
			graphConfig = mergeJSONObjects(req.GraphDefaults, graphConfig)
		}

		if createErr := h.createEmailAccount(ctx, candidate.Address, candidate.Provider, graphConfig, status); createErr != nil {
			h.logger.Warn("batch email register: failed to create account",
				zap.String("address", candidate.Address),
				zap.Error(createErr),
			)
			failed++
			continue
		}
		created++
	}

	h.completeTrackedJob(ctx, payload.JobID, map[string]any{
		"requested": req.Count,
		"generated": len(prepared.Candidates),
		"created":   created,
		"failed":    failed,
		"canceled":  canceled,
	}, batchSummaryStatus(canceled, failed))

	h.logger.Info("batch email register finished",
		zap.Int("requested", req.Count),
		zap.Int("generated", len(prepared.Candidates)),
		zap.Int("created", created),
		zap.Int("failed", failed),
		zap.Bool("canceled", canceled),
	)
	return nil
}

func (h *BatchHandler) ProcessEmailBatchImportGraph(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseBatchEmailImportGraphPayload(t)
	if err != nil {
		h.logger.Warn("invalid batch email graph import payload", zap.Error(err))
		return asynq.SkipRetry
	}
	if !h.beginTrackedJob(ctx, payload.JobID) {
		return nil
	}

	if h.isTrackedJobCanceled(ctx, payload.JobID) {
		h.completeTrackedJob(ctx, payload.JobID, map[string]any{
			"total":    len(payload.Request.Rows),
			"canceled": true,
		}, jobStatusCanceled)
		return nil
	}

	result, execErr := service.ExecuteBatchImportGraphTask(ctx, h.emailRepo, payload.Request, func() bool {
		return h.isTrackedJobCanceled(ctx, payload.JobID)
	})
	status := batchSummaryStatus(false, result.Failed)
	if execErr != nil {
		status = jobStatusFailed
	}
	if h.isTrackedJobCanceled(ctx, payload.JobID) {
		status = jobStatusCanceled
	}

	summary := map[string]any{
		"total":    result.Total,
		"created":  result.Created,
		"failed":   result.Failed,
		"failures": result.Failures,
		"error": func() string {
			if execErr == nil {
				return ""
			}
			return execErr.Error()
		}(),
	}
	if status == jobStatusCanceled {
		summary["canceled"] = true
	}
	h.completeTrackedJob(ctx, payload.JobID, summary, status)

	if execErr != nil {
		h.logger.Error("batch email graph import failed", zap.Error(execErr))
		return nil
	}

	h.logger.Info("batch email graph import finished",
		zap.Int("total", result.Total),
		zap.Int("created", result.Created),
		zap.Int("failed", result.Failed),
		zap.Int("failure_details", len(result.Failures)),
	)
	return nil
}

func (h *BatchHandler) createEmailAccount(
	ctx context.Context,
	address string,
	provider string,
	graphConfig json.RawMessage,
	status int16,
) error {
	if h.emailRepo == nil {
		return errors.New("email repository is not configured")
	}

	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" {
		return errors.New("address is required")
	}
	parsed, err := mail.ParseAddress(trimmedAddress)
	if err != nil {
		return fmt.Errorf("address must be a valid email address: %w", err)
	}
	normalizedAddress := strings.ToLower(strings.TrimSpace(parsed.Address))

	if !isJSONObject(graphConfig) {
		return errors.New("graph_config must be a valid JSON object")
	}
	if status != 0 && status != 1 {
		return errors.New("status must be 0 or 1")
	}

	item := &model.EmailAccount{
		Address:     normalizedAddress,
		Provider:    normalizeEmailProvider(provider, normalizedAddress),
		GraphConfig: normalizeJSON(graphConfig, "{}"),
		Status:      status,
	}
	if err := h.emailRepo.Create(ctx, item); err != nil {
		return err
	}
	return nil
}

func (h *BatchHandler) beginTrackedJob(ctx context.Context, jobID uint64) bool {
	if jobID == 0 || h.jobRepo == nil {
		return true
	}

	job, err := h.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Warn("failed to load batch job", zap.Uint64("job_id", jobID), zap.Error(err))
		}
		return true
	}
	if job.Status == jobStatusCanceled || job.Status == jobStatusDone || job.Status == jobStatusFailed {
		h.logger.Info("batch job skipped: already finished", zap.Uint64("job_id", jobID), zap.Int16("status", job.Status))
		return false
	}
	if _, err := h.jobRepo.UpdateStatus(ctx, jobID, jobStatusRunning); err != nil {
		h.logger.Warn("failed to update batch job status to running", zap.Uint64("job_id", jobID), zap.Error(err))
	}
	return true
}

func (h *BatchHandler) isTrackedJobCanceled(ctx context.Context, jobID uint64) bool {
	if jobID == 0 || h.jobRepo == nil {
		return false
	}
	job, err := h.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Warn("failed to check batch job status", zap.Uint64("job_id", jobID), zap.Error(err))
		}
		return false
	}
	return job.Status == jobStatusCanceled
}

func (h *BatchHandler) completeTrackedJob(ctx context.Context, jobID uint64, result any, status int16) {
	if jobID == 0 {
		return
	}
	h.persistTrackedJobRun(ctx, jobID, result, status)
	if h.jobRepo == nil {
		return
	}
	if _, err := h.jobRepo.UpdateStatus(ctx, jobID, status); err != nil {
		h.logger.Warn("failed to update batch job status", zap.Uint64("job_id", jobID), zap.Int16("status", status), zap.Error(err))
	}
}

func (h *BatchHandler) persistTrackedJobRun(ctx context.Context, jobID uint64, result any, status int16) {
	if h.jobRunRepo == nil {
		return
	}

	startedAt := time.Now().UTC()
	endedAt := startedAt

	var resultJSON []byte
	if result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			h.logger.Warn("failed to marshal batch job result", zap.Uint64("job_id", jobID), zap.Error(err))
		} else {
			resultJSON = raw
		}
	}

	errorCode := ""
	errorMessage := ""
	if status == jobStatusFailed {
		errorCode = "BATCH_TASK_FAILED"
		errorMessage = "batch task finished with failures"
	}
	if status == jobStatusCanceled {
		errorCode = "CANCELED"
		errorMessage = "task canceled"
	}

	jobRun := &model.JobRun{
		JobID:        jobID,
		WorkerID:     h.workerID,
		Attempt:      1,
		Result:       resultJSON,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		StartedAt:    startedAt,
		EndedAt:      &endedAt,
	}
	if err := h.jobRunRepo.Create(ctx, jobRun); err != nil {
		h.logger.Warn("failed to persist batch job run", zap.Uint64("job_id", jobID), zap.Error(err))
	}
}

func batchSummaryStatus(canceled bool, failed int) int16 {
	if canceled {
		return jobStatusCanceled
	}
	if failed > 0 {
		return jobStatusFailed
	}
	return jobStatusDone
}

func normalizeJSON(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return value
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	return obj != nil
}

func normalizeEmailProvider(provider string, address string) string {
	value := strings.ToLower(strings.TrimSpace(provider))
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

func mergeJSONObjects(base, overlay json.RawMessage) json.RawMessage {
	if len(overlay) == 0 {
		return base
	}
	var baseMap map[string]json.RawMessage
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return base
	}
	var overlayMap map[string]json.RawMessage
	if err := json.Unmarshal(overlay, &overlayMap); err != nil {
		return base
	}
	if baseMap == nil {
		baseMap = make(map[string]json.RawMessage)
	}
	for k, v := range overlayMap {
		if _, exists := baseMap[k]; !exists {
			baseMap[k] = v
		}
	}
	merged, err := json.Marshal(baseMap)
	if err != nil {
		return base
	}
	return merged
}
