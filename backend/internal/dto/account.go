package dto

import (
    "encoding/json"
    "time"
)

type ListAccountsQuery struct {
    TypeKey string `form:"type_key" binding:"omitempty"`
}

type CreateAccountRequest struct {
    TypeKey    string          `json:"type_key" binding:"required"`
    Identifier string          `json:"identifier" binding:"required"`
    Status     int16           `json:"status,omitempty" binding:"omitempty"`
    Tags       []string        `json:"tags,omitempty" binding:"omitempty"`
    Spec       json.RawMessage `json:"spec" binding:"required"`
}

type PatchAccountRequest struct {
    Status *int16           `json:"status,omitempty" binding:"omitempty"`
    Tags   []string         `json:"tags,omitempty" binding:"omitempty"`
    Spec   *json.RawMessage `json:"spec,omitempty" binding:"omitempty"`
}

type BatchPatchAccountRequest struct {
    IDs    []uint64 `json:"ids" binding:"required,min=1"`
    Status *int16   `json:"status,omitempty" binding:"omitempty"`
    Tags   []string `json:"tags,omitempty" binding:"omitempty"`
}

type BatchDeleteAccountRequest struct {
    IDs []uint64 `json:"ids" binding:"required,min=1"`
}

type AccountResponse struct {
    ID         uint64          `json:"id"`
    TypeKey    string          `json:"type_key"`
    Identifier string          `json:"identifier"`
    Status     int16           `json:"status"`
    Tags       []string        `json:"tags,omitempty"`
    Spec       json.RawMessage `json:"spec"`
    CreatedAt  time.Time       `json:"created_at"`
    UpdatedAt  time.Time       `json:"updated_at"`
}
