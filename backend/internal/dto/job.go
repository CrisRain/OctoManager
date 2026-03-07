package dto

import (
	"encoding/json"
	"time"
)

type CreateJobRequest struct {
	TypeKey   string          `json:"type_key" binding:"required"`
	ActionKey string          `json:"action_key" binding:"required"`
	Selector  json.RawMessage `json:"selector" binding:"required"`
	Params    json.RawMessage `json:"params" binding:"required"`
}

type PatchJobRequest struct {
	TypeKey   *string          `json:"type_key,omitempty" binding:"omitempty"`
	ActionKey *string          `json:"action_key,omitempty" binding:"omitempty"`
	Selector  *json.RawMessage `json:"selector,omitempty" binding:"omitempty"`
	Params    *json.RawMessage `json:"params,omitempty" binding:"omitempty"`
	Status    *int16           `json:"status,omitempty" binding:"omitempty,oneof=0 1 2 3 4"`
}

type JobResponse struct {
	ID        uint64          `json:"id"`
	TypeKey   string          `json:"type_key"`
	ActionKey string          `json:"action_key"`
	Selector  json.RawMessage `json:"selector"`
	Params    json.RawMessage `json:"params"`
	Status    int16           `json:"status"`
	LastRun   *JobRunResponse `json:"last_run,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
