package task

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hibiken/asynq"

	"octomanger/backend/internal/dto"
)

type Producer struct {
	client *asynq.Client
}

func NewProducer(client *asynq.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) EnqueueDispatchJob(ctx context.Context, jobID uint64) error {
	if p == nil || p.client == nil {
		return errors.New("asynq client is not configured")
	}
	payload, err := json.Marshal(DispatchJobPayload{JobID: jobID})
	if err != nil {
		return err
	}
	taskItem := asynq.NewTask(TypeDispatchJob, payload)
	_, err = p.client.EnqueueContext(ctx, taskItem,
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	return err
}

func (p *Producer) EnqueueBatchAccountPatch(ctx context.Context, jobID uint64, req dto.BatchPatchAccountRequest) (string, error) {
	return p.enqueueTaskID(ctx, TypeBatchAccountPatch, BatchAccountPatchPayload{JobID: jobID, Request: req})
}

func (p *Producer) EnqueueBatchAccountDelete(ctx context.Context, jobID uint64, req dto.BatchDeleteAccountRequest) (string, error) {
	return p.enqueueTaskID(ctx, TypeBatchAccountDelete, BatchAccountDeletePayload{JobID: jobID, Request: req})
}

func (p *Producer) EnqueueBatchEmailDelete(ctx context.Context, jobID uint64, req dto.BatchEmailAccountIDsRequest) (string, error) {
	return p.enqueueTaskID(ctx, TypeBatchEmailDelete, BatchEmailDeletePayload{JobID: jobID, Request: req})
}

func (p *Producer) EnqueueBatchEmailVerify(ctx context.Context, jobID uint64, req dto.BatchEmailAccountIDsRequest) (string, error) {
	return p.enqueueTaskID(ctx, TypeBatchEmailVerify, BatchEmailVerifyPayload{JobID: jobID, Request: req})
}

func (p *Producer) EnqueueBatchEmailRegister(ctx context.Context, jobID uint64, req dto.BatchRegisterEmailRequest) (string, error) {
	return p.enqueueTaskID(ctx, TypeBatchEmailRegister, BatchEmailRegisterPayload{JobID: jobID, Request: req})
}

func (p *Producer) EnqueueBatchEmailImportGraph(ctx context.Context, jobID uint64, req dto.BatchImportGraphEmailTaskRequest) (string, error) {
	return p.enqueueTaskID(ctx, TypeBatchEmailImportGraph, BatchEmailImportGraphPayload{JobID: jobID, Request: req})
}

func (p *Producer) enqueueTaskID(ctx context.Context, taskType string, payload any) (string, error) {
	if p == nil || p.client == nil {
		return "", errors.New("asynq client is not configured")
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	taskItem := asynq.NewTask(taskType, rawPayload)
	info, err := p.client.EnqueueContext(ctx, taskItem,
		asynq.Queue("default"),
		asynq.MaxRetry(1),
		asynq.Timeout(10*time.Minute),
	)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}
