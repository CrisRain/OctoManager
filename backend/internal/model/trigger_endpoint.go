package model

import "encoding/json"

type TriggerEndpoint struct {
	BaseModel
	Name            string          `gorm:"type:text;not null" json:"name"`
	Slug            string          `gorm:"type:text;not null;uniqueIndex" json:"slug"`
	TypeKey         string          `gorm:"type:text;not null" json:"type_key"`
	ActionKey       string          `gorm:"type:text;not null" json:"action_key"`
	ExecutionMode   string          `gorm:"column:execution_mode;type:text;not null;default:'async'" json:"mode"`
	DefaultSelector json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"default_selector"`
	DefaultParams   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"default_params"`
	TokenHash       string          `gorm:"type:text;not null" json:"-"`
	TokenPrefix     string          `gorm:"type:text;not null" json:"token_prefix"`
	Enabled         bool            `gorm:"not null;default:true" json:"enabled"`
}

func (TriggerEndpoint) TableName() string {
	return "trigger_endpoints"
}
