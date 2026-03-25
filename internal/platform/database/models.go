package database

import (
	"encoding/json"
	"time"
)

type AccountTypeModel struct {
	ID               int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Key              string          `gorm:"column:key;type:text;not null;uniqueIndex"`
	Name             string          `gorm:"column:name;type:text;not null"`
	Category         string          `gorm:"column:category;type:text;not null;default:generic"`
	SchemaJSON       json.RawMessage `gorm:"column:schema_json;type:jsonb;not null;default:'{}'"`
	CapabilitiesJSON json.RawMessage `gorm:"column:capabilities_json;type:jsonb;not null;default:'{}'"`
	CreatedAt        time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (AccountTypeModel) TableName() string {
	return "account_types"
}

type AccountModel struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement"`
	AccountTypeID *int64          `gorm:"column:account_type_id;index:idx_accounts_type_identifier,unique"`
	Identifier    string          `gorm:"column:identifier;type:text;not null;index:idx_accounts_type_identifier,unique"`
	SpecJSON      json.RawMessage `gorm:"column:spec_json;type:jsonb;not null;default:'{}'"`
	Status        string          `gorm:"column:status;type:text;not null;default:active"`
	TagsJSON      json.RawMessage `gorm:"column:tags_json;type:jsonb;not null;default:'[]'"`
	CreatedAt     time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (AccountModel) TableName() string {
	return "accounts"
}

type EmailAccountModel struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Provider   string          `gorm:"column:provider;type:text;not null"`
	Address    string          `gorm:"column:address;type:text;not null;uniqueIndex"`
	Status     string          `gorm:"column:status;type:text;not null;default:active"`
	ConfigJSON json.RawMessage `gorm:"column:config_json;type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (EmailAccountModel) TableName() string {
	return "email_accounts"
}

type JobDefinitionModel struct {
	ID        int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Key       string          `gorm:"column:key;type:text;not null;uniqueIndex"`
	Name      string          `gorm:"column:name;type:text;not null"`
	PluginKey string          `gorm:"column:plugin_key;type:text;not null"`
	Action    string          `gorm:"column:action;type:text;not null"`
	InputJSON json.RawMessage `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	Enabled   bool            `gorm:"column:enabled;not null;default:true"`
	CreatedAt time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (JobDefinitionModel) TableName() string {
	return "job_definitions"
}

type ScheduleModel struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	JobDefinitionID int64      `gorm:"column:job_definition_id;not null;uniqueIndex"`
	CronExpression  string     `gorm:"column:cron_expression;type:text;not null"`
	Timezone        string     `gorm:"column:timezone;type:text;not null;default:UTC"`
	NextRunAt       *time.Time `gorm:"column:next_run_at"`
	LeaseOwner      *string    `gorm:"column:lease_owner;type:text"`
	LeaseUntil      *time.Time `gorm:"column:lease_until"`
	Enabled         bool       `gorm:"column:enabled;not null;default:true"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ScheduleModel) TableName() string {
	return "schedules"
}

type JobExecutionModel struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement"`
	JobDefinitionID int64           `gorm:"column:job_definition_id;not null;index:idx_job_executions_def_id_created,priority:1"`
	InputJSON       json.RawMessage `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	Status          string          `gorm:"column:status;type:text;not null;index:idx_job_executions_status_created_at,priority:1"`
	RequestedBy     string          `gorm:"column:requested_by;type:text;not null;default:''"`
	Source          string          `gorm:"column:source;type:text;not null;default:manual"`
	WorkerID        *string         `gorm:"column:worker_id;type:text"`
	Summary         *string         `gorm:"column:summary;type:text"`
	ResultJSON      json.RawMessage `gorm:"column:result_json;type:jsonb;not null;default:'{}'"`
	ErrorMessage    *string         `gorm:"column:error_message;type:text"`
	StartedAt       *time.Time      `gorm:"column:started_at"`
	FinishedAt      *time.Time      `gorm:"column:finished_at"`
	CreatedAt       time.Time       `gorm:"column:created_at;not null;autoCreateTime;index:idx_job_executions_status_created_at,priority:2;sort:desc;index:idx_job_executions_def_id_created,priority:2,sort:desc"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (JobExecutionModel) TableName() string {
	return "job_executions"
}

type JobLogModel struct {
	ID             int64           `gorm:"column:id;primaryKey;autoIncrement"`
	JobExecutionID int64           `gorm:"column:job_execution_id;not null;index:idx_job_logs_execution_id_id,priority:1"`
	Stream         string          `gorm:"column:stream;type:text;not null"`
	EventType      string          `gorm:"column:event_type;type:text;not null"`
	Message        string          `gorm:"column:message;type:text;not null"`
	PayloadJSON    json.RawMessage `gorm:"column:payload_json;type:jsonb;not null;default:'{}'"`
	CreatedAt      time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
}

func (JobLogModel) TableName() string {
	return "job_logs"
}

type TriggerModel struct {
	ID               int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Key              string          `gorm:"column:key;type:text;not null;uniqueIndex"`
	Name             string          `gorm:"column:name;type:text;not null"`
	JobDefinitionID  int64           `gorm:"column:job_definition_id;not null"`
	Mode             string          `gorm:"column:mode;type:text;not null;default:async"`
	DefaultInputJSON json.RawMessage `gorm:"column:default_input_json;type:jsonb;not null;default:'{}'"`
	TokenHash        string          `gorm:"column:token_hash;type:text;not null"`
	TokenPrefix      string          `gorm:"column:token_prefix;type:text;not null"`
	Enabled          bool            `gorm:"column:enabled;not null;default:true"`
	CreatedAt        time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (TriggerModel) TableName() string {
	return "triggers"
}

type AgentModel struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Name            string          `gorm:"column:name;type:text;not null"`
	PluginKey       string          `gorm:"column:plugin_key;type:text;not null"`
	Action          string          `gorm:"column:action;type:text;not null"`
	InputJSON       json.RawMessage `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	DesiredState    string          `gorm:"column:desired_state;type:text;not null;default:stopped"`
	RuntimeState    string          `gorm:"column:runtime_state;type:text;not null;default:idle"`
	LastError       *string         `gorm:"column:last_error;type:text"`
	LastHeartbeatAt *time.Time      `gorm:"column:last_heartbeat_at"`
	CreatedAt       time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (AgentModel) TableName() string {
	return "agents"
}

type AgentLogModel struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID     int64           `gorm:"column:agent_id;not null;index:idx_agent_logs_agent_id_id,priority:1"`
	EventType   string          `gorm:"column:event_type;type:text;not null"`
	Message     string          `gorm:"column:message;type:text;not null"`
	PayloadJSON json.RawMessage `gorm:"column:payload_json;type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
}

func (AgentLogModel) TableName() string {
	return "agent_logs"
}

type SystemConfigModel struct {
	Key       string          `gorm:"column:key;primaryKey;type:text"`
	ValueJSON json.RawMessage `gorm:"column:value_json;type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time       `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (SystemConfigModel) TableName() string {
	return "system_configs"
}
