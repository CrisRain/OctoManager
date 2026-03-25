package jobpostgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	jobdomain "octomanger/internal/domains/jobs/domain"
	"octomanger/internal/platform/dbutil"
)

const (
	jobExecutionCacheTTL = 3 * time.Second
	jobLogCacheTTL       = time.Hour
	jobLogRetentionLimit = 200
)

func (r Repository) executionCacheKey(executionID int64) string {
	return fmt.Sprintf("jobs:execution:%d", executionID)
}

func (r Repository) executionsCacheKey() string {
	return "jobs:executions:list"
}

func (r Repository) executionLogsCacheKey(executionID int64) string {
	return fmt.Sprintf("jobs:execution:%d:logs", executionID)
}

func (r Repository) readExecutionCache(ctx context.Context, executionID int64) (*jobdomain.JobExecution, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.executionCacheKey(executionID)).Bytes()
	if err != nil {
		return nil, false
	}
	var item jobdomain.JobExecution
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, false
	}
	return &item, true
}

func (r Repository) writeExecutionCache(ctx context.Context, item *jobdomain.JobExecution) {
	if r.rdb == nil || item == nil {
		return
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.executionCacheKey(item.ID), raw, jobExecutionCacheTTL).Err()
}

func (r Repository) invalidateExecutionCache(ctx context.Context, executionID int64) {
	if r.rdb == nil {
		return
	}
	_ = r.rdb.Del(ctx, r.executionCacheKey(executionID), r.executionsCacheKey()).Err()
}

func (r Repository) readExecutionsCache(ctx context.Context) ([]jobdomain.JobExecution, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.executionsCacheKey()).Bytes()
	if err != nil {
		return nil, false
	}
	var items []jobdomain.JobExecution
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) writeExecutionsCache(ctx context.Context, items []jobdomain.JobExecution) {
	if r.rdb == nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.executionsCacheKey(), raw, jobExecutionCacheTTL).Err()
}

func (r Repository) readExecutionLogsCache(ctx context.Context, executionID int64) ([]jobdomain.JobLog, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.executionLogsCacheKey(executionID)).Bytes()
	if err != nil {
		return nil, false
	}
	var items []jobdomain.JobLog
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) refreshExecutionLogsCache(ctx context.Context, executionID int64) {
	if r.rdb == nil {
		return
	}
	items, err := r.loadRecentExecutionLogs(ctx, executionID)
	if err != nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.executionLogsCacheKey(executionID), raw, jobLogCacheTTL).Err()
}

func (r Repository) loadRecentExecutionLogs(ctx context.Context, executionID int64) ([]jobdomain.JobLog, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, job_execution_id, stream, event_type, message, payload_json, created_at
		FROM (
			SELECT id, job_execution_id, stream, event_type, message, payload_json, created_at
			FROM job_logs
			WHERE job_execution_id = $1
			ORDER BY id DESC
			LIMIT $2
		) recent
		ORDER BY id ASC`, executionID, jobLogRetentionLimit).Rows()
	if err != nil {
		return nil, fmt.Errorf("load recent execution logs: %w", err)
	}
	defer rows.Close()

	items := make([]jobdomain.JobLog, 0)
	for rows.Next() {
		var item jobdomain.JobLog
		var payloadJSON []byte
		if err := rows.Scan(&item.ID, &item.JobExecutionID, &item.Stream, &item.EventType, &item.Message, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan execution log: %w", err)
		}
		item.Payload = dbutil.DecodeJSONMap(payloadJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) trimExecutionLogs(ctx context.Context, executionID int64) error {
	result := r.db.WithContext(ctx).Exec(`
		DELETE FROM job_logs
		WHERE job_execution_id = $1
		  AND id < COALESCE((
			SELECT id
			FROM job_logs
			WHERE job_execution_id = $1
			ORDER BY id DESC
			OFFSET $2
			LIMIT 1
		  ), 0)`, executionID, jobLogRetentionLimit-1)
	if result.Error != nil {
		return fmt.Errorf("trim execution logs: %w", result.Error)
	}
	return nil
}
