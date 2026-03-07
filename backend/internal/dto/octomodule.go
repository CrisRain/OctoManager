package dto

import (
    "encoding/json"
    "time"
)

type OctoModuleInfoResponse struct {
    TypeKey      string          `json:"type_key"`
    Category     string          `json:"category"`
    ScriptPath   string          `json:"script_path"`
    ModuleDir    string          `json:"module_dir"`
    EntryFile    string          `json:"entry_file"`
    Source       string          `json:"source"`
    Exists       bool            `json:"exists"`
    ScriptConfig json.RawMessage `json:"script_config,omitempty"`
    Error        string          `json:"error,omitempty"`
}

type OctoModuleSyncItemResponse struct {
    TypeKey    string          `json:"type_key"`
    Category   string          `json:"category"`
    ScriptPath string          `json:"script_path"`
    Source     string          `json:"source"`
    Exists     bool            `json:"exists"`
    Created    bool            `json:"created"`
    Error      string          `json:"error,omitempty"`
    Config     json.RawMessage `json:"script_config,omitempty"`
}

type OctoModuleSyncResponse struct {
    Total    int                      `json:"total"`
    Created  int                      `json:"created"`
    Existing int                      `json:"existing"`
    Failed   int                      `json:"failed"`
    Items    []OctoModuleSyncItemResponse `json:"items"`
}

type OctoModuleDryRunAccountRequest struct {
    Identifier string         `json:"identifier" binding:"required"`
    Spec       map[string]any `json:"spec,omitempty" binding:"omitempty"`
}

type OctoModuleDryRunContextRequest struct {
    RequestID string `json:"request_id,omitempty" binding:"omitempty"`
}

type OctoModuleDryRunRequest struct {
    Action  string                         `json:"action" binding:"required"`
    Account OctoModuleDryRunAccountRequest `json:"account" binding:"required"`
    Params  map[string]any                 `json:"params,omitempty" binding:"omitempty"`
    Context OctoModuleDryRunContextRequest `json:"context,omitempty" binding:"omitempty"`
}

type OctoModuleOutputResponse struct {
    Status       string         `json:"status"`
    Result       map[string]any `json:"result,omitempty"`
    ErrorCode    string         `json:"error_code,omitempty"`
    ErrorMessage string         `json:"error_message,omitempty"`
}

type OctoModuleDryRunResponse struct {
    Module OctoModuleInfoResponse `json:"module"`
    Output OctoModuleOutputResponse `json:"output"`
}

type OctoModuleEnsureResponse struct {
    Module  OctoModuleInfoResponse `json:"module"`
    Created bool                   `json:"created"`
}

type OctoModuleScriptResponse struct {
    Content string `json:"content"`
}

type UpdateOctoModuleScriptRequest struct {
    Content string `json:"content" binding:"required"`
}

type JobRunResponse struct {
    ID           uint64          `json:"id"`
    JobID        uint64          `json:"job_id"`
    JobTypeKey   string          `json:"job_type_key"`
    JobActionKey string          `json:"job_action_key"`
    AccountID    *uint64         `json:"account_id,omitempty"`
    WorkerID     string          `json:"worker_id"`
    Attempt      int             `json:"attempt"`
    Result       json.RawMessage `json:"result,omitempty"`
    ErrorCode    string          `json:"error_code,omitempty"`
    ErrorMessage string          `json:"error_message,omitempty"`
    StartedAt    time.Time       `json:"started_at"`
    EndedAt      *time.Time      `json:"ended_at,omitempty"`
}

type OctoModuleRunHistoryResponse struct {
    Items  []JobRunResponse `json:"items"`
    Total  int64            `json:"total"`
    Limit  int              `json:"limit"`
    Offset int              `json:"offset"`
}

type OctoModuleFileInfo struct {
    Name    string `json:"name"`
    Size    int64  `json:"size"`
    IsEntry bool   `json:"is_entry"`
}

type ListOctoModuleFilesResponse struct {
    ModuleDir string               `json:"module_dir"`
    EntryFile string               `json:"entry_file"`
    Files     []OctoModuleFileInfo `json:"files"`
}

type UpdateOctoModuleFileRequest struct {
    Content string `json:"content" binding:"required"`
}

type VenvInfoResponse struct {
    Exists              bool   `json:"exists"`
    Dir                 string `json:"dir"`
    PythonPath          string `json:"python_path"`
    HasRequirements     bool   `json:"has_requirements"`
    RequirementsContent string `json:"requirements_content,omitempty"`
}

type InstallDepsRequest struct {
    Packages            []string `json:"packages,omitempty"`
    FromRequirements    bool     `json:"from_requirements,omitempty"`
    RequirementsContent string   `json:"requirements_content,omitempty"`
}

type InstallDepsResponse struct {
    Success bool   `json:"success"`
    Output  string `json:"output"`
}
