package model

import "time"

const (
    ApiKeyRoleAdmin   = "admin"
    ApiKeyRoleWebhook = "webhook"

    ApiKeyWebhookScopeAll = "*"
)

type ApiKey struct {
    BaseModel
    Name         string     `gorm:"type:text;not null" json:"name"`
    KeyHash      string     `gorm:"type:text;not null;uniqueIndex" json:"-"`
    KeyPrefix    string     `gorm:"type:text;not null" json:"key_prefix"`
    Role         string     `gorm:"type:text;not null;default:'admin'" json:"role"`
    WebhookScope string     `gorm:"type:text;not null;default:'*'" json:"webhook_scope,omitempty"`
    Enabled      bool       `gorm:"not null;default:true" json:"enabled"`
    LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}

func (ApiKey) TableName() string {
    return "api_keys"
}
