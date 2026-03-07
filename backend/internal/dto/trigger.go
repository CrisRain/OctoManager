package dto

import (
	"encoding/json"
	"time"
)

type CreateTriggerRequest struct {
	Name            string          `json:"name" binding:"required"`
	Slug            string          `json:"slug" binding:"required"`
	TypeKey         string          `json:"type_key" binding:"required"`
	ActionKey       string          `json:"action_key" binding:"required"`
	Mode            string          `json:"mode,omitempty" binding:"omitempty,oneof=sync async"`
	DefaultSelector json.RawMessage `json:"default_selector,omitempty" binding:"omitempty"`
	DefaultParams   json.RawMessage `json:"default_params,omitempty" binding:"omitempty"`
}

type PatchTriggerRequest struct {
	Name            *string          `json:"name,omitempty" binding:"omitempty"`
	TypeKey         *string          `json:"type_key,omitempty" binding:"omitempty"`
	ActionKey       *string          `json:"action_key,omitempty" binding:"omitempty"`
	Mode            *string          `json:"mode,omitempty" binding:"omitempty,oneof=sync async"`
	DefaultSelector *json.RawMessage `json:"default_selector,omitempty" binding:"omitempty"`
	DefaultParams   *json.RawMessage `json:"default_params,omitempty" binding:"omitempty"`
	Enabled         *bool            `json:"enabled,omitempty" binding:"omitempty"`
}

type FireTriggerRequest struct {
	Mode        string          `json:"mode,omitempty" binding:"omitempty,oneof=sync async"`
	Selector    json.RawMessage `json:"selector,omitempty" binding:"omitempty"`
	ExtraParams json.RawMessage `json:"extra_params,omitempty" binding:"omitempty"`
}

type TriggerEndpointResponse struct {
	ID              uint64          `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	TypeKey         string          `json:"type_key"`
	ActionKey       string          `json:"action_key"`
	Mode            string          `json:"mode"`
	DefaultSelector json.RawMessage `json:"default_selector"`
	DefaultParams   json.RawMessage `json:"default_params"`
	TokenPrefix     string          `json:"token_prefix"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CreateTriggerResponse struct {
	Endpoint TriggerEndpointResponse `json:"endpoint"`
	RawToken string                  `json:"raw_token,omitempty"`
}

type FireTriggerResponse struct {
	Endpoint TriggerEndpointResponse `json:"endpoint"`
	Mode     string                  `json:"mode"`
	Queued   bool                    `json:"queued"`
	Input    TriggerExecutionInput   `json:"input"`
	Job      *JobResponse            `json:"job,omitempty"`
	Output   *TriggerExecutionOutput `json:"output,omitempty"`
}

type TriggerExecutionInput struct {
	TypeKey   string          `json:"type_key"`
	ActionKey string          `json:"action_key"`
	Selector  json.RawMessage `json:"selector"`
	Params    json.RawMessage `json:"params"`
}

type TriggerExecutionOutput struct {
	JobStatus         int16                           `json:"job_status"`
	MatchedAccounts   int                             `json:"matched_accounts"`
	ProcessedAccounts int                             `json:"processed_accounts"`
	ErrorCode         string                          `json:"error_code,omitempty"`
	ErrorMessage      string                          `json:"error_message,omitempty"`
	Results           []TriggerExecutionAccountResult `json:"results"`
}

type TriggerExecutionAccountResult struct {
	RunID        uint64                   `json:"run_id,omitempty"`
	AccountID    uint64                   `json:"account_id"`
	Identifier   string                   `json:"identifier"`
	Status       string                   `json:"status"`
	Result       map[string]any           `json:"result,omitempty"`
	ErrorCode    string                   `json:"error_code,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
	Session      *TriggerExecutionSession `json:"session,omitempty"`
	StartedAt    time.Time                `json:"started_at"`
	EndedAt      *time.Time               `json:"ended_at,omitempty"`
}

type TriggerExecutionSession struct {
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	ExpiresAt string         `json:"expires_at,omitempty"`
}
