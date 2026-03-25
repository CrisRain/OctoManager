package triggerdomain

import "time"

type Trigger struct {
	ID              int64          `json:"id"`
	Key             string         `json:"key"`
	Name            string         `json:"name"`
	JobDefinitionID int64          `json:"job_definition_id"`
	Mode            string         `json:"mode"`
	DefaultInput    map[string]any `json:"default_input"`
	TokenPrefix     string         `json:"token_prefix"`
	Enabled         bool           `json:"enabled"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type CreateInput struct {
	Key             string         `json:"key"`
	Name            string         `json:"name"`
	JobDefinitionID int64          `json:"job_definition_id"`
	Mode            string         `json:"mode"`
	DefaultInput    map[string]any `json:"default_input"`
	Enabled         bool           `json:"enabled"`
}

type CreateResult struct {
	Trigger       Trigger `json:"trigger"`
	DeliveryToken string  `json:"delivery_token"`
}

type PatchTriggerInput struct {
	Name            *string        `json:"name,omitempty"`
	JobDefinitionID *int64         `json:"job_definition_id,omitempty"`
	Mode            *string        `json:"mode,omitempty"`
	DefaultInput    map[string]any `json:"default_input,omitempty"`
	Enabled         *bool          `json:"enabled,omitempty"`
}

type FireResult struct {
	Mode        string         `json:"mode"`
	TriggerKey  string         `json:"trigger_key"`
	ExecutionID *int64         `json:"execution_id,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Events      []any          `json:"events,omitempty"`
}
