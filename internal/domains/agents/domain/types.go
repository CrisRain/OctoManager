package agentdomain

import "time"

type Agent struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	PluginKey       string         `json:"plugin_key"`
	Action          string         `json:"action"`
	Input           map[string]any `json:"input"`
	DesiredState    string         `json:"desired_state"`
	RuntimeState    string         `json:"runtime_state"`
	LastError       string         `json:"last_error,omitempty"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type AgentLog struct {
	ID        int64          `json:"id"`
	AgentID   int64          `json:"agent_id"`
	EventType string         `json:"event_type"`
	Message   string         `json:"message"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// AgentStatus is the lightweight status snapshot served by the REST status endpoint.
// It is cached in Redis to avoid polling the DB on every frontend request.
type AgentStatus struct {
	ID              int64      `json:"id"`
	RuntimeState    string     `json:"runtime_state"`
	DesiredState    string     `json:"desired_state"`
	LastError       string     `json:"last_error,omitempty"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentLogEntry is the write model for a single agent log row (used by batch inserts).
type AgentLogEntry struct {
	AgentID   int64
	EventType string
	Message   string
	Payload   map[string]any
}

type CreateAgentInput struct {
	Name      string         `json:"name"`
	PluginKey string         `json:"plugin_key"`
	Action    string         `json:"action"`
	Input     map[string]any `json:"input"`
}

type PatchAgentInput struct {
	Name  *string        `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}
