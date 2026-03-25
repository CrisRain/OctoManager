package jobdomain

import "time"

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

type Schedule struct {
	ID              int64      `json:"id"`
	JobDefinitionID int64      `json:"job_definition_id"`
	CronExpression  string     `json:"cron_expression"`
	Timezone        string     `json:"timezone"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LeaseOwner      string     `json:"lease_owner,omitempty"`
	LeaseUntil      *time.Time `json:"lease_until,omitempty"`
	Enabled         bool       `json:"enabled"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type JobDefinition struct {
	ID        int64          `json:"id"`
	Key       string         `json:"key"`
	Name      string         `json:"name"`
	PluginKey string         `json:"plugin_key"`
	Action    string         `json:"action"`
	Input     map[string]any `json:"input"`
	Enabled   bool           `json:"enabled"`
	Schedule  *Schedule      `json:"schedule,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type JobExecution struct {
	ID              int64          `json:"id"`
	JobDefinitionID int64          `json:"job_definition_id"`
	DefinitionKey   string         `json:"definition_key"`
	DefinitionName  string         `json:"definition_name"`
	PluginKey       string         `json:"plugin_key"`
	Action          string         `json:"action"`
	Status          string         `json:"status"`
	Input           map[string]any `json:"input"`
	RequestedBy     string         `json:"requested_by"`
	Source          string         `json:"source"`
	WorkerID        string         `json:"worker_id,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Result          map[string]any `json:"result,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type JobLog struct {
	ID             int64          `json:"id"`
	JobExecutionID int64          `json:"job_execution_id"`
	Stream         string         `json:"stream"`
	EventType      string         `json:"event_type"`
	Message        string         `json:"message"`
	Payload        map[string]any `json:"payload,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// JobLogEntry is the write model for a single log row (used by batch inserts).
type JobLogEntry struct {
	ExecutionID int64
	Stream      string
	EventType   string
	Message     string
	Payload     map[string]any
}

type CreateDefinitionInput struct {
	Key       string         `json:"key"`
	Name      string         `json:"name"`
	PluginKey string         `json:"plugin_key"`
	Action    string         `json:"action"`
	Input     map[string]any `json:"input"`
	Schedule  *ScheduleInput `json:"schedule,omitempty"`
}

type ScheduleInput struct {
	CronExpression string `json:"cron_expression"`
	Timezone       string `json:"timezone,omitempty"`
	Enabled        bool   `json:"enabled"`
}

type PatchDefinitionInput struct {
	Name     *string        `json:"name,omitempty"`
	Input    map[string]any `json:"input,omitempty"`
	Enabled  *bool          `json:"enabled,omitempty"`
	Schedule *ScheduleInput `json:"schedule,omitempty"`
}
