package model

import "time"

type AccountSession struct {
    BaseModel
    AccountID        uint64     `gorm:"type:bigint;not null;index:idx_account_sessions_account_id" json:"account_id"`
    SessionType      int16      `gorm:"type:smallint;not null" json:"session_type"`
    EncryptedPayload []byte     `gorm:"type:bytea;not null" json:"-"`
    ExpiresAt        *time.Time `gorm:"index:idx_account_sessions_expires_at" json:"expires_at,omitempty"`
    State            int16      `gorm:"type:smallint;not null;default:0" json:"state"`
}

func (AccountSession) TableName() string {
    return "account_sessions"
}
