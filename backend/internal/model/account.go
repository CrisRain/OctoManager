package model

import "encoding/json"

type Account struct {
	BaseModel
	TypeKey    string          `gorm:"type:text;not null;index:idx_accounts_type;uniqueIndex:uidx_accounts_type_identifier" json:"type_key"`
	Identifier string          `gorm:"type:text;not null;uniqueIndex:uidx_accounts_type_identifier" json:"identifier"`
	Status     int16           `gorm:"type:smallint;not null;index:idx_accounts_status" json:"status"`
	Tags       StringArray     `gorm:"type:text[];not null;default:'{}'" json:"tags,omitempty"`
	Spec       json.RawMessage `gorm:"type:jsonb;not null;default:'{}';index:idx_accounts_spec, type:gin" json:"spec"`
}

func (Account) TableName() string {
	return "accounts"
}
