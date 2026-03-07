package task

import (
	"encoding/json"
	"errors"

	"github.com/hibiken/asynq"

	"octomanger/backend/internal/dto"
)

const (
	TypeDispatchJob           = "job:dispatch"
	TypeBatchAccountPatch     = "batch:account:patch"
	TypeBatchAccountDelete    = "batch:account:delete"
	TypeBatchEmailDelete      = "batch:email:delete"
	TypeBatchEmailVerify      = "batch:email:verify"
	TypeBatchEmailRegister    = "batch:email:register"
	TypeBatchEmailImportGraph = "batch:email:import_graph"
)

type DispatchJobPayload struct {
	JobID uint64 `json:"job_id"`
}

type BatchAccountPatchPayload struct {
	JobID   uint64                       `json:"job_id,omitempty"`
	Request dto.BatchPatchAccountRequest `json:"request"`
}

type BatchAccountDeletePayload struct {
	JobID   uint64                        `json:"job_id,omitempty"`
	Request dto.BatchDeleteAccountRequest `json:"request"`
}

type BatchEmailDeletePayload struct {
	JobID   uint64                          `json:"job_id,omitempty"`
	Request dto.BatchEmailAccountIDsRequest `json:"request"`
}

type BatchEmailVerifyPayload struct {
	JobID   uint64                          `json:"job_id,omitempty"`
	Request dto.BatchEmailAccountIDsRequest `json:"request"`
}

type BatchEmailRegisterPayload struct {
	JobID   uint64                        `json:"job_id,omitempty"`
	Request dto.BatchRegisterEmailRequest `json:"request"`
}

type BatchEmailImportGraphPayload struct {
	JobID   uint64                               `json:"job_id,omitempty"`
	Request dto.BatchImportGraphEmailTaskRequest `json:"request"`
}

func ParseDispatchJobPayload(task *asynq.Task) (DispatchJobPayload, error) {
	if task == nil {
		return DispatchJobPayload{}, errors.New("task is nil")
	}
	var payload DispatchJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return DispatchJobPayload{}, err
	}
	if payload.JobID == 0 {
		return DispatchJobPayload{}, errors.New("job_id is required")
	}
	return payload, nil
}

func ParseBatchAccountPatchPayload(task *asynq.Task) (BatchAccountPatchPayload, error) {
	if task == nil {
		return BatchAccountPatchPayload{}, errors.New("task is nil")
	}
	var payload BatchAccountPatchPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return BatchAccountPatchPayload{}, err
	}
	if len(payload.Request.IDs) == 0 {
		return BatchAccountPatchPayload{}, errors.New("ids is required")
	}
	if payload.Request.Status == nil && payload.Request.Tags == nil {
		return BatchAccountPatchPayload{}, errors.New("status or tags is required")
	}
	return payload, nil
}

func ParseBatchAccountDeletePayload(task *asynq.Task) (BatchAccountDeletePayload, error) {
	if task == nil {
		return BatchAccountDeletePayload{}, errors.New("task is nil")
	}
	var payload BatchAccountDeletePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return BatchAccountDeletePayload{}, err
	}
	if len(payload.Request.IDs) == 0 {
		return BatchAccountDeletePayload{}, errors.New("ids is required")
	}
	return payload, nil
}

func ParseBatchEmailDeletePayload(task *asynq.Task) (BatchEmailDeletePayload, error) {
	if task == nil {
		return BatchEmailDeletePayload{}, errors.New("task is nil")
	}
	var payload BatchEmailDeletePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return BatchEmailDeletePayload{}, err
	}
	if len(payload.Request.IDs) == 0 {
		return BatchEmailDeletePayload{}, errors.New("ids is required")
	}
	return payload, nil
}

func ParseBatchEmailVerifyPayload(task *asynq.Task) (BatchEmailVerifyPayload, error) {
	if task == nil {
		return BatchEmailVerifyPayload{}, errors.New("task is nil")
	}
	var payload BatchEmailVerifyPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return BatchEmailVerifyPayload{}, err
	}
	if len(payload.Request.IDs) == 0 {
		return BatchEmailVerifyPayload{}, errors.New("ids is required")
	}
	return payload, nil
}

func ParseBatchEmailRegisterPayload(task *asynq.Task) (BatchEmailRegisterPayload, error) {
	if task == nil {
		return BatchEmailRegisterPayload{}, errors.New("task is nil")
	}
	var payload BatchEmailRegisterPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return BatchEmailRegisterPayload{}, err
	}
	if payload.Request.Count <= 0 {
		return BatchEmailRegisterPayload{}, errors.New("count must be > 0")
	}
	return payload, nil
}

func ParseBatchEmailImportGraphPayload(task *asynq.Task) (BatchEmailImportGraphPayload, error) {
	if task == nil {
		return BatchEmailImportGraphPayload{}, errors.New("task is nil")
	}
	var payload BatchEmailImportGraphPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return BatchEmailImportGraphPayload{}, err
	}
	if len(payload.Request.Rows) == 0 {
		return BatchEmailImportGraphPayload{}, errors.New("rows is required")
	}
	return payload, nil
}
