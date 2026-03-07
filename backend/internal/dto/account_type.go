package dto

import (
    "encoding/json"
    "time"
)

type CreateAccountTypeRequest struct {
    Key          string          `json:"key" binding:"required"`
    Name         string          `json:"name" binding:"required"`
    Category     string          `json:"category" binding:"required,oneof=system email generic"`
    Schema       json.RawMessage `json:"schema" binding:"required"`
    Capabilities json.RawMessage `json:"capabilities" binding:"required"`
    ScriptConfig json.RawMessage `json:"script_config,omitempty" binding:"omitempty"`
}

type PatchAccountTypeRequest struct {
    Name         *string         `json:"name,omitempty" binding:"omitempty"`
    Category     *string         `json:"category,omitempty" binding:"omitempty,oneof=system email generic"`
    Schema       *json.RawMessage `json:"schema,omitempty" binding:"omitempty"`
    Capabilities *json.RawMessage `json:"capabilities,omitempty" binding:"omitempty"`
    ScriptConfig *json.RawMessage `json:"script_config,omitempty" binding:"omitempty"`
}

type AccountTypeResponse struct {
    ID           uint64          `json:"id"`
    Key          string          `json:"key"`
    Name         string          `json:"name"`
    Category     string          `json:"category"`
    Schema       json.RawMessage `json:"schema"`
    Capabilities json.RawMessage `json:"capabilities"`
    ScriptConfig json.RawMessage `json:"script_config,omitempty"`
    Version      int             `json:"version"`
    CreatedAt    time.Time       `json:"created_at"`
    UpdatedAt    time.Time       `json:"updated_at"`
}
