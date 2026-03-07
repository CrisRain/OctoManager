package adapter

import "context"

type Account struct {
	ID         string         `json:"id"`
	Identifier string         `json:"identifier"`
	Spec       map[string]any `json:"spec"`
}

type ActionRequest struct {
	TenantID     string         `json:"tenant_id"`
	RequestID    string         `json:"request_id"`
	TypeKey      string         `json:"type_key"`
	Action       string         `json:"action"`
	ModuleScript string         `json:"module_script,omitempty"`
	Params       map[string]any `json:"params"`
	Account      Account        `json:"account"`
}

type Session struct {
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	ExpiresAt string         `json:"expires_at,omitempty"`
}

type Result struct {
	Status  string         `json:"status"`
	Result  map[string]any `json:"result,omitempty"`
	Session *Session       `json:"session,omitempty"`
}

type Adapter interface {
	TypeKey() string
	ValidateSpec(spec map[string]any) error
	ExecuteAction(ctx context.Context, request ActionRequest) (Result, error)
}
