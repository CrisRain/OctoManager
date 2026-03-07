package model

import (
    "encoding/json"
    "time"
)

type JobRun struct {
    BaseID
    JobID        uint64          `gorm:"type:bigint;not null;index:idx_job_runs_job_id" json:"job_id"`
    AccountID    *uint64         `gorm:"type:bigint;index:idx_job_runs_account_id" json:"account_id,omitempty"`
    WorkerID     string          `gorm:"type:text;not null;index:idx_job_runs_worker_id" json:"worker_id"`
    Attempt      int             `gorm:"not null;default:1" json:"attempt"`
    Result       json.RawMessage `gorm:"type:jsonb" json:"result,omitempty"`
    ErrorCode    string          `gorm:"type:text" json:"error_code,omitempty"`
    ErrorMessage string          `gorm:"type:text" json:"error_message,omitempty"`
    StartedAt    time.Time       `gorm:"not null" json:"started_at"`
    EndedAt      *time.Time      `json:"ended_at,omitempty"`
}

func (JobRun) TableName() string {
    return "job_runs"
}
