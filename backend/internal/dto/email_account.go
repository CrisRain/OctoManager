package dto

import (
	"encoding/json"
	"time"
)

type CreateEmailAccountRequest struct {
	Address     string          `json:"address" binding:"required"`
	Provider    string          `json:"provider,omitempty" binding:"omitempty"`
	GraphConfig json.RawMessage `json:"graph_config" binding:"required"`
	Status      int16           `json:"status,omitempty" binding:"omitempty,oneof=0 1"`
}

type PatchEmailAccountRequest struct {
	Address     *string          `json:"address,omitempty" binding:"omitempty"`
	Provider    *string          `json:"provider,omitempty" binding:"omitempty"`
	GraphConfig *json.RawMessage `json:"graph_config,omitempty" binding:"omitempty"`
	Status      *int16           `json:"status,omitempty" binding:"omitempty,oneof=0 1"`
}

type BatchEmailAccountIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"required,min=1"`
}

type BatchImportGraphEmailRequest struct {
	Content         string   `json:"content" binding:"required"`
	DefaultClientID string   `json:"default_client_id,omitempty" binding:"omitempty"`
	Tenant          string   `json:"tenant,omitempty" binding:"omitempty"`
	Scope           []string `json:"scope,omitempty" binding:"omitempty"`
	Mailbox         string   `json:"mailbox,omitempty" binding:"omitempty"`
	GraphBaseURL    string   `json:"graph_base_url,omitempty" binding:"omitempty"`
	Status          int16    `json:"status,omitempty" binding:"omitempty,oneof=0 1"`
}

type BatchImportGraphEmailFailure struct {
	Line    int    `json:"line"`
	Address string `json:"address,omitempty"`
	Error   string `json:"error"`
}

type BatchImportGraphEmailTaskRow struct {
	Line         int    `json:"line"`
	Address      string `json:"address"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
}

type BatchImportGraphEmailTaskRequest struct {
	Rows         []BatchImportGraphEmailTaskRow `json:"rows"`
	Tenant       string                         `json:"tenant,omitempty"`
	Scope        []string                       `json:"scope,omitempty"`
	Mailbox      string                         `json:"mailbox,omitempty"`
	GraphBaseURL string                         `json:"graph_base_url,omitempty"`
	Status       int16                          `json:"status,omitempty"`
}

type BatchImportGraphEmailResponse struct {
	Total    int                            `json:"total"`
	Accepted int                            `json:"accepted"`
	Skipped  int                            `json:"skipped"`
	Failures []BatchImportGraphEmailFailure `json:"failures"`
	Queued   bool                           `json:"queued,omitempty"`
	TaskID   string                         `json:"task_id,omitempty"`
	JobID    uint64                         `json:"job_id,omitempty"`
}

type BatchRegisterEmailRequest struct {
	Operator      string          `json:"operator,omitempty" binding:"omitempty"`
	Provider      string          `json:"provider,omitempty" binding:"omitempty"`
	Count         int             `json:"count" binding:"required,min=1,max=200"`
	Prefix        string          `json:"prefix,omitempty" binding:"omitempty"`
	Domain        string          `json:"domain,omitempty" binding:"omitempty"`
	StartIndex    int             `json:"start_index,omitempty" binding:"omitempty,min=0"`
	Status        int16           `json:"status,omitempty" binding:"omitempty,oneof=0 1"`
	Options       map[string]any  `json:"options,omitempty" binding:"omitempty"`
	GraphDefaults json.RawMessage `json:"graph_defaults,omitempty" binding:"omitempty"`
}

type BatchRegisterEmailFailure struct {
	Index   int    `json:"index"`
	Address string `json:"address,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BatchRegisterEmailResponse struct {
	Requested int                         `json:"requested"`
	Generated int                         `json:"generated"`
	Created   int                         `json:"created"`
	Failed    int                         `json:"failed"`
	Provider  string                      `json:"provider,omitempty"`
	Accounts  []EmailAccountResponse      `json:"accounts"`
	Failures  []BatchRegisterEmailFailure `json:"failures"`
	Queued    bool                        `json:"queued,omitempty"`
	TaskID    string                      `json:"task_id,omitempty"`
	JobID     uint64                      `json:"job_id,omitempty"`
}

type PreviewEmailRequest struct {
	GraphConfig json.RawMessage `json:"graph_config" binding:"required"`
	Mailbox     string          `json:"mailbox,omitempty" binding:"omitempty"`
	Reference   string          `json:"reference,omitempty" binding:"omitempty"`
	Pattern     string          `json:"pattern,omitempty" binding:"omitempty"`
}

type EmailConfigSummary struct {
	Host                string   `json:"host,omitempty"`
	Port                int      `json:"port,omitempty"`
	SSL                 bool     `json:"ssl"`
	StartTLS            bool     `json:"starttls"`
	Username            string   `json:"username,omitempty"`
	TokenUsername       string   `json:"token_username,omitempty"`
	AuthMethod          string   `json:"auth_method,omitempty"`
	Tenant              string   `json:"tenant,omitempty"`
	Mailbox             string   `json:"mailbox,omitempty"`
	Scope               []string `json:"scope,omitempty"`
	TokenExpiresAt      string   `json:"token_expires_at,omitempty"`
	AccessTokenPresent  bool     `json:"access_token_present"`
	RefreshTokenPresent bool     `json:"refresh_token_present"`
	ClientIDPresent     bool     `json:"client_id_present"`
	ClientSecretPresent bool     `json:"client_secret_present"`
}

type EmailAccountResponse struct {
	ID           uint64              `json:"id"`
	Address      string              `json:"address"`
	Provider     string              `json:"provider,omitempty"`
	Status       int16               `json:"status"`
	GraphSummary *EmailConfigSummary `json:"graph_summary,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}
