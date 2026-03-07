package model

import "encoding/json"

type EmailAccount struct {
	BaseModel
	Address           string          `gorm:"type:text;not null;uniqueIndex:uidx_email_accounts_address" json:"address"`
	Provider          string          `gorm:"type:text" json:"provider,omitempty"`
	EncryptedPassword []byte          `gorm:"type:bytea" json:"-"`
	GraphConfig       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"graph_config,omitempty"`
	Status            int16           `gorm:"type:smallint;not null;default:0;index:idx_email_accounts_status" json:"status"`
}

func (EmailAccount) TableName() string {
	return "email_accounts"
}
