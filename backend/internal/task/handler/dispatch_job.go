package taskhandler

import (
	"context"
	"errors"
	"strings"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/service"
	"octomanger/backend/internal/task"
	"octomanger/backend/internal/worker/bridge"
)

const (
	jobStatusQueued   int16 = service.JobStatusQueued
	jobStatusRunning  int16 = service.JobStatusRunning
	jobStatusDone     int16 = service.JobStatusDone
	jobStatusFailed   int16 = service.JobStatusFailed
	jobStatusCanceled int16 = service.JobStatusCanceled
)

type JobHandlerOptions struct {
	Logger             *zap.Logger
	JobRepo            repository.JobRepository
	AccountRepo        repository.AccountRepository
	AccountTypeRepo    repository.AccountTypeRepository
	JobRunRepo         repository.JobRunRepository
	AccountSessionRepo repository.AccountSessionRepository
	PythonBridge       bridge.PythonBridge
	ModuleDir          string
	WorkerID           string
}

type JobHandler struct {
	logger   *zap.Logger
	executor service.JobExecutor
}

func NewJobHandler(opts JobHandlerOptions) *JobHandler {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	executor := service.NewJobExecutor(service.JobExecutorOptions{
		Logger:             logger,
		JobRepo:            opts.JobRepo,
		AccountRepo:        opts.AccountRepo,
		AccountTypeRepo:    opts.AccountTypeRepo,
		JobRunRepo:         opts.JobRunRepo,
		AccountSessionRepo: opts.AccountSessionRepo,
		PythonBridge:       opts.PythonBridge,
		ModuleDir:          strings.TrimSpace(opts.ModuleDir),
		WorkerID:           strings.TrimSpace(opts.WorkerID),
	})

	return &JobHandler{
		logger:   logger,
		executor: executor,
	}
}

func (h *JobHandler) ProcessDispatchJob(ctx context.Context, t *asynq.Task) error {
	payload, err := task.ParseDispatchJobPayload(t)
	if err != nil {
		h.logger.Warn("invalid dispatch payload", zap.Error(err))
		return asynq.SkipRetry
	}

	if h.executor == nil {
		h.logger.Error("dispatch executor is not configured", zap.Uint64("job_id", payload.JobID))
		return asynq.SkipRetry
	}

	summary, err := h.executor.ExecuteJob(ctx, payload.JobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Warn("dispatch job not found", zap.Uint64("job_id", payload.JobID))
			return asynq.SkipRetry
		}
		h.logger.Error("failed to execute dispatch job", zap.Uint64("job_id", payload.JobID), zap.Error(err))
		return err
	}
	if summary == nil {
		h.logger.Warn("dispatch executor returned empty summary", zap.Uint64("job_id", payload.JobID))
		return asynq.SkipRetry
	}

	fields := []zap.Field{
		zap.Uint64("job_id", summary.JobID),
		zap.Int16("status", summary.Status),
		zap.Int("matched_accounts", summary.MatchedAccounts),
		zap.Int("processed_accounts", summary.ProcessedAccounts),
	}
	if summary.ErrorCode != "" {
		fields = append(fields, zap.String("error_code", summary.ErrorCode))
	}
	if summary.ErrorMessage != "" {
		fields = append(fields, zap.String("error_message", summary.ErrorMessage))
	}

	switch summary.Status {
	case service.JobStatusDone:
		h.logger.Info("dispatch finished", fields...)
		return nil
	case service.JobStatusCanceled:
		h.logger.Info("dispatch canceled", fields...)
		return nil
	case service.JobStatusFailed:
		h.logger.Warn("dispatch failed", fields...)
		return asynq.SkipRetry
	default:
		h.logger.Info("dispatch finished with intermediate status", fields...)
		return nil
	}
}
