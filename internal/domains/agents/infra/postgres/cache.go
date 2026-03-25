package agentpostgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentdomain "octomanger/internal/domains/agents/domain"
	"octomanger/internal/platform/dbutil"
)

const (
	agentListCacheTTL = 3 * time.Second
	agentLogCacheTTL  = time.Hour
	agentLogLimit     = 200
)

func (r Repository) agentsCacheKey() string {
	return "agents:list"
}

func (r Repository) agentLogsCacheKey(agentID int64) string {
	return fmt.Sprintf("agents:%d:logs", agentID)
}

func (r Repository) readAgentsCache(ctx context.Context) ([]agentdomain.Agent, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.agentsCacheKey()).Bytes()
	if err != nil {
		return nil, false
	}
	var items []agentdomain.Agent
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) writeAgentsCache(ctx context.Context, items []agentdomain.Agent) {
	if r.rdb == nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.agentsCacheKey(), raw, agentListCacheTTL).Err()
}

func (r Repository) invalidateAgentsCache(ctx context.Context) {
	if r.rdb == nil {
		return
	}
	_ = r.rdb.Del(ctx, r.agentsCacheKey()).Err()
}

func (r Repository) readAgentLogsCache(ctx context.Context, agentID int64) ([]agentdomain.AgentLog, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.agentLogsCacheKey(agentID)).Bytes()
	if err != nil {
		return nil, false
	}
	var items []agentdomain.AgentLog
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) refreshAgentLogsCache(ctx context.Context, agentID int64) {
	if r.rdb == nil {
		return
	}
	items, err := r.loadRecentAgentLogs(ctx, agentID)
	if err != nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.agentLogsCacheKey(agentID), raw, agentLogCacheTTL).Err()
}

func (r Repository) loadRecentAgentLogs(ctx context.Context, agentID int64) ([]agentdomain.AgentLog, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, agent_id, event_type, message, payload_json, created_at
		FROM (
			SELECT id, agent_id, event_type, message, payload_json, created_at
			FROM agent_logs
			WHERE agent_id = $1 AND (event_type != '' OR message != '')
			ORDER BY id DESC
			LIMIT $2
		) recent
		ORDER BY id ASC`, agentID, agentLogLimit).Rows()
	if err != nil {
		return nil, fmt.Errorf("load recent agent logs: %w", err)
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
	return items, rows.Err()
}

func (r Repository) trimAgentLogs(ctx context.Context, agentID int64) error {
	result := r.db.WithContext(ctx).Exec(`
		DELETE FROM agent_logs
		WHERE agent_id = $1
		  AND id < COALESCE((
			SELECT id
			FROM agent_logs
			WHERE agent_id = $1
			ORDER BY id DESC
			OFFSET $2
			LIMIT 1
		  ), 0)`, agentID, agentLogLimit-1)
	if result.Error != nil {
		return fmt.Errorf("trim agent logs: %w", result.Error)
	}
	return nil
}
