package model

import "encoding/json"

type AccountType struct {
    BaseModel
    Key          string          `gorm:"type:text;not null;uniqueIndex" json:"key"`
    Name         string          `gorm:"type:text;not null" json:"name"`
    Category     string          `gorm:"type:text;not null" json:"category"`
    Schema       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"schema"`
    Capabilities json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"capabilities"`
    ScriptConfig json.RawMessage `gorm:"type:jsonb" json:"script_config,omitempty"`
    Version      int             `gorm:"not null;default:1" json:"version"`
}

func (AccountType) TableName() string {
    return "account_types"
}
