package model

import "encoding/json"

type SystemConfig struct {
    BaseTimestamps
    Key         string          `gorm:"type:text;primaryKey" json:"key"`
    Value       json.RawMessage `gorm:"type:jsonb;not null" json:"value"`
    IsCritical  bool            `gorm:"not null;default:false" json:"is_critical"`
    Description string          `gorm:"type:text;not null;default:''" json:"description"`
}

func (SystemConfig) TableName() string {
    return "system_configs"
}
