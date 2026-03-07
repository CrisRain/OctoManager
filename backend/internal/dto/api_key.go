package dto

import "time"

type CreateApiKeyRequest struct {
    Name         string `json:"name" binding:"required"`
    Role         string `json:"role"`
    WebhookScope string `json:"webhook_scope"`
}

type SetApiKeyEnabledRequest struct {
    Enabled bool `json:"enabled" binding:"required"`
}

type ApiKeyResponse struct {
    ID           uint64     `json:"id"`
    Name         string     `json:"name"`
    KeyPrefix    string     `json:"key_prefix"`
    Role         string     `json:"role"`
    WebhookScope string     `json:"webhook_scope,omitempty"`
    Enabled      bool       `json:"enabled"`
    LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}

type CreateApiKeyResponse struct {
    ApiKey ApiKeyResponse `json:"api_key"`
    RawKey string         `json:"raw_key"`
}
