package service

import (
	"context"
	"encoding/json"
	"errors"

	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
)

const (
	asyncJobTypeSystem = "system"

	asyncJobActionBatchAccountPatch     = "batch_account_patch"
	asyncJobActionBatchAccountDelete    = "batch_account_delete"
	asyncJobActionBatchEmailDelete      = "batch_email_delete"
	asyncJobActionBatchEmailVerify      = "batch_email_verify"
	asyncJobActionBatchEmailRegister    = "batch_email_register"
	asyncJobActionBatchEmailImportGraph = "batch_email_import_graph"
)

type asyncJobEnqueueFunc func(jobID uint64) (string, error)

type asyncJobSpec struct {
	TypeKey   string
	ActionKey string
	Selector  any
	Params    any
}

func createAndEnqueueAsyncJob(
	ctx context.Context,
	jobRepo repository.JobRepository,
	dispatcher JobDispatcher,
	spec asyncJobSpec,
	enqueue asyncJobEnqueueFunc,
) (*model.Job, string, error) {
	if jobRepo == nil {
		return nil, "", errors.New("job repository is not configured")
	}
	if dispatcher == nil {
		return nil, "", errors.New("job dispatcher is not configured")
	}

	selector, err := marshalAsyncJobMetadata(spec.Selector)
	if err != nil {
		return nil, "", err
	}
	params, err := marshalAsyncJobMetadata(spec.Params)
	if err != nil {
		return nil, "", err
	}

	item := &model.Job{
		TypeKey:   spec.TypeKey,
		ActionKey: spec.ActionKey,
		Selector:  selector,
		Params:    params,
		Status:    jobStatusQueued,
	}
	if err := jobRepo.Create(ctx, item); err != nil {
		return nil, "", err
	}

	taskID, err := enqueue(item.ID)
	if err != nil {
		_, _ = jobRepo.UpdateStatus(ctx, item.ID, jobStatusFailed)
		return item, "", err
	}
	return item, taskID, nil
}

func marshalAsyncJobMetadata(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("{}"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if !isJSONObject(raw) {
		return nil, errors.New("async job metadata must be a JSON object")
	}
	return raw, nil
}
