package agentpostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	agentdomain "octomanger/internal/domains/agents/domain"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("agent not found")

type Repository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func New(db *gorm.DB, rdb ...*redis.Client) Repository {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return Repository{db: db, rdb: client}
}

func (r Repository) List(ctx context.Context) ([]agentdomain.Agent, error) {
	if items, ok := r.readAgentsCache(ctx); ok {
		return items, nil
	}
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, plugin_key, action, input_json, desired_state, runtime_state,
		       COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at
		FROM agents ORDER BY created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	items, err := scanAgents(rows)
	if err != nil {
		return nil, err
	}
	r.writeAgentsCache(ctx, items)
	return items, nil
}

func (r Repository) ListDesiredRunning(ctx context.Context) ([]agentdomain.Agent, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, plugin_key, action, input_json, desired_state, runtime_state,
		       COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at
		FROM agents WHERE desired_state = 'running' ORDER BY id ASC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list runnable agents: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r Repository) Create(ctx context.Context, input agentdomain.CreateAgentInput) (*agentdomain.Agent, error) {
	inputJSON, err := json.Marshal(dbutil.NormalizeMap(input.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal agent input: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO agents (name, plugin_key, action, input_json, desired_state, runtime_state)
		VALUES ($1, $2, $3, $4, 'stopped', 'idle')
		RETURNING id, name, plugin_key, action, input_json, desired_state, runtime_state,
		          COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at`,
		input.Name, input.PluginKey, input.Action, inputJSON,
	).Row()
	item, err := scanAgent(row)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	r.invalidateAgentsCache(ctx)
	return &item, nil
}

func (r Repository) Get(ctx context.Context, agentID int64) (*agentdomain.Agent, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT id, name, plugin_key, action, input_json, desired_state, runtime_state,
		       COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at
		FROM agents WHERE id = $1`, agentID).Row()
	item, err := scanAgent(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, agentID int64, input agentdomain.PatchAgentInput) (*agentdomain.Agent, error) {
	current, err := r.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.Input != nil {
		current.Input = input.Input
	}

	inputJSON, err := json.Marshal(dbutil.NormalizeMap(current.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal agent input: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		UPDATE agents SET name = $2, input_json = $3, updated_at = NOW() WHERE id = $1
		RETURNING id, name, plugin_key, action, input_json, desired_state, runtime_state,
		          COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at`,
		agentID, current.Name, inputJSON,
	).Row()
	item, err := scanAgent(row)
	if err != nil {
		return nil, fmt.Errorf("patch agent: %w", err)
	}
	r.invalidateAgentsCache(ctx)
	return &item, nil
}

func (r Repository) SetDesiredState(ctx context.Context, agentID int64, desiredState string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE agents SET desired_state = $2, updated_at = NOW() WHERE id = $1`,
		agentID, desiredState,
	)
	if result.Error != nil {
		return fmt.Errorf("update agent desired state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r Repository) UpdateRuntimeState(ctx context.Context, agentID int64, runtimeState, lastError string, heartbeat *time.Time) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE agents
		SET runtime_state = $2, last_error = $3, last_heartbeat_at = $4, updated_at = NOW()
		WHERE id = $1`,
		agentID, runtimeState, lastError, heartbeat,
	)
	if result.Error != nil {
		return fmt.Errorf("update agent runtime state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.invalidateAgentsCache(ctx)
	return nil
}

func (r Repository) Delete(ctx context.Context, agentID int64) error {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM agents WHERE id = $1`, agentID)
	if result.Error != nil {
		return fmt.Errorf("delete agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.invalidateAgentsCache(ctx)
	return nil
}

func (r Repository) AppendLog(ctx context.Context, agentID int64, eventType, message string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(dbutil.NormalizeMap(payload))
	if err != nil {
		return fmt.Errorf("marshal agent log payload: %w", err)
	}
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO agent_logs (agent_id, event_type, message, payload_json)
		VALUES ($1, $2, $3, $4)`,
		agentID, eventType, message, payloadJSON,
	)
	if result.Error != nil {
		return fmt.Errorf("append agent log: %w", result.Error)
	}
	if err := r.trimAgentLogs(ctx, agentID); err != nil {
		return err
	}
	r.refreshAgentLogsCache(ctx, agentID)
	return nil
}

func (r Repository) AppendLogBatch(ctx context.Context, entries []agentdomain.AgentLogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	const cols = 4
	args := make([]any, 0, len(entries)*cols)
	placeholders := make([]string, len(entries))
	for i, e := range entries {
		base := i * cols
		placeholders[i] = fmt.Sprintf("($%d,$%d,$%d,$%d)", base+1, base+2, base+3, base+4)
		p, _ := json.Marshal(dbutil.NormalizeMap(e.Payload))
		args = append(args, e.AgentID, e.EventType, e.Message, p)
	}
	query := "INSERT INTO agent_logs (agent_id, event_type, message, payload_json) VALUES " +
		strings.Join(placeholders, ",")
	if result := r.db.WithContext(ctx).Exec(query, args...); result.Error != nil {
		return fmt.Errorf("append agent log batch: %w", result.Error)
	}
	seen := make(map[int64]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.AgentID]; ok {
			continue
		}
		seen[entry.AgentID] = struct{}{}
		if err := r.trimAgentLogs(ctx, entry.AgentID); err != nil {
			return err
		}
		r.refreshAgentLogsCache(ctx, entry.AgentID)
	}
	return nil
}

func (r Repository) ListLogsAfter(ctx context.Context, agentID, afterID int64) ([]agentdomain.AgentLog, error) {
	if items, ok := r.readAgentLogsCache(ctx, agentID); ok {
		filtered := make([]agentdomain.AgentLog, 0, len(items))
		for _, item := range items {
			if item.ID > afterID {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) > agentLogLimit {
			return filtered[:agentLogLimit], nil
		}
		return filtered, nil
	}
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, agent_id, event_type, message, payload_json, created_at
		FROM agent_logs
		WHERE agent_id = $1 AND id > $2 AND (event_type != '' OR message != '')
		ORDER BY id ASC
		LIMIT 200`, agentID, afterID).Rows()
	if err != nil {
		return nil, fmt.Errorf("list agent logs: %w", err)
	}
	defer rows.Close()

	items := make([]agentdomain.AgentLog, 0)
	for rows.Next() {
		var item agentdomain.AgentLog
		var payloadJSON []byte
		if err := rows.Scan(&item.ID, &item.AgentID, &item.EventType, &item.Message, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent log: %w", err)
		}
		item.Payload = dbutil.DecodeJSONMap(payloadJSON)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.refreshAgentLogsCache(ctx, agentID)
	return items, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAgents(rows *sql.Rows) ([]agentdomain.Agent, error) {
	items := make([]agentdomain.Agent, 0)
	for rows.Next() {
		item, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAgent(row scanner) (agentdomain.Agent, error) {
	var item agentdomain.Agent
	var inputJSON []byte
	var lastHeartbeat sql.NullTime
	if err := row.Scan(
		&item.ID, &item.Name, &item.PluginKey, &item.Action, &inputJSON,
		&item.DesiredState, &item.RuntimeState, &item.LastError,
		&lastHeartbeat, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agentdomain.Agent{}, ErrNotFound
		}
		return agentdomain.Agent{}, fmt.Errorf("scan agent: %w", err)
	}
	item.Input = dbutil.DecodeJSONMap(inputJSON)
	if lastHeartbeat.Valid {
		t := lastHeartbeat.Time
		item.LastHeartbeatAt = &t
	}
	return item, nil
}
