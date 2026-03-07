package service

import (
	"context"

	"octomanger/backend/internal/dto"
)

// JobDispatcher defines the async dispatch capability for created jobs.
type JobDispatcher interface {
	EnqueueDispatchJob(ctx context.Context, jobID uint64) error
	EnqueueBatchAccountPatch(ctx context.Context, jobID uint64, req dto.BatchPatchAccountRequest) (string, error)
	EnqueueBatchAccountDelete(ctx context.Context, jobID uint64, req dto.BatchDeleteAccountRequest) (string, error)
	EnqueueBatchEmailDelete(ctx context.Context, jobID uint64, req dto.BatchEmailAccountIDsRequest) (string, error)
	EnqueueBatchEmailVerify(ctx context.Context, jobID uint64, req dto.BatchEmailAccountIDsRequest) (string, error)
	EnqueueBatchEmailRegister(ctx context.Context, jobID uint64, req dto.BatchRegisterEmailRequest) (string, error)
	EnqueueBatchEmailImportGraph(ctx context.Context, jobID uint64, req dto.BatchImportGraphEmailTaskRequest) (string, error)
}

// JobExecutor runs a job immediately in-process using the same dispatch semantics as the worker.
type JobExecutor interface {
	ExecuteJob(ctx context.Context, jobID uint64) (*JobExecutionSummary, error)
}
