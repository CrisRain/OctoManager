package model

import "encoding/json"

type Job struct {
    BaseModel
    TypeKey   string          `gorm:"type:text;not null;index:idx_jobs_type_action,priority:1" json:"type_key"`
    ActionKey string          `gorm:"type:text;not null;index:idx_jobs_type_action,priority:2" json:"action_key"`
    Selector  json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"selector"`
    Params    json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"params"`
    Status    int16           `gorm:"type:smallint;not null;default:0;index:idx_jobs_status" json:"status"`
}

func (Job) TableName() string {
    return "jobs"
}
