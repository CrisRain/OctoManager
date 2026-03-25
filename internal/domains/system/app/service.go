package systemapp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	plugindomain "octomanger/internal/domains/plugins/domain"
)

// pluginLister is the narrow interface systemapp needs from the plugin backend.
type pluginLister interface {
	List(ctx context.Context) ([]plugindomain.Plugin, error)
}

type Status struct {
	Now         time.Time `json:"now"`
	DatabaseOK  bool      `json:"database_ok"`
	PluginCount int       `json:"plugin_count"`
}

// DashboardSummary is returned by GET /api/v2/dashboard and replaces 8 parallel
// frontend requests with a single round-trip. Counts are produced by cheap SQL
// COUNT queries; recent executions fetch the latest 6 rows only.
type DashboardSummary struct {
	PluginCount        int               `json:"pluginCount"`
	AccountTypeCount   int               `json:"accountTypeCount"`
	AccountCount       int               `json:"accountCount"`
	EmailAccountCount  int               `json:"emailAccountCount"`
	JobDefinitionCount int               `json:"jobDefinitionCount"`
	JobExecutionCount  int               `json:"jobExecutionCount"`
	TriggerCount       int               `json:"triggerCount"`
	AgentCount         int               `json:"agentCount"`
	RecentExecutions   []RecentExecution `json:"recentExecutions"`
}

// RecentExecution is a lightweight projection of job_executions for the
// dashboard table. It mirrors the fields the frontend currently reads.
type RecentExecution struct {
	ID             int64  `json:"id"`
	DefinitionName string `json:"definition_name"`
	PluginKey      string `json:"plugin_key"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	WorkerID       string `json:"worker_id"`
}

type Service struct {
	db      *gorm.DB
	plugins pluginLister
	rdb     *redis.Client
}

func New(db *gorm.DB, plugins pluginLister, rdb ...*redis.Client) Service {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return Service{db: db, plugins: plugins, rdb: client}
}

func (s Service) Status(ctx context.Context) (Status, error) {
	if item, ok := s.readStatusCache(ctx); ok {
		return item, nil
	}
	plugins, err := s.plugins.List(ctx)
	if err != nil {
		return Status{}, err
	}

	item := Status{
		Now:         time.Now().UTC(),
		DatabaseOK:  s.db != nil,
		PluginCount: len(plugins),
	}
	s.writeStatusCache(ctx, item)
	return item, nil
}

// DashboardSummary returns entity counts and the 6 most recent executions in a
// single DB round-trip (one query per table — all cheap COUNT/LIMIT queries).
func (s Service) DashboardSummary(ctx context.Context) (DashboardSummary, error) {
	if item, ok := s.readDashboardCache(ctx); ok {
		return item, nil
	}
	type countRow struct{ N int64 }

	countOf := func(table string) (int, error) {
		var row countRow
		if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) AS n FROM " + table).Scan(&row).Error; err != nil {
			return 0, err
		}
		return int(row.N), nil
	}

	pluginList, _ := s.plugins.List(ctx) // plugins live in-memory; no separate table

	accountTypeCount, err := countOf("account_types")
	if err != nil {
		return DashboardSummary{}, err
	}

	accountCount, err := countOf("accounts")
	if err != nil {
		return DashboardSummary{}, err
	}

	emailAccountCount, err := countOf("email_accounts")
	if err != nil {
		return DashboardSummary{}, err
	}

	jobDefinitionCount, err := countOf("job_definitions")
	if err != nil {
		return DashboardSummary{}, err
	}

	jobExecutionCount, err := countOf("job_executions")
	if err != nil {
		return DashboardSummary{}, err
	}

	triggerCount, err := countOf("triggers")
	if err != nil {
		return DashboardSummary{}, err
	}

	agentCount, err := countOf("agents")
	if err != nil {
		return DashboardSummary{}, err
	}

	// Fetch the 6 most recent executions — a lightweight projection, not the full scan.
	rows, err := s.db.WithContext(ctx).Raw(`
		SELECT e.id, d.name, d.plugin_key, d.action, e.status, COALESCE(e.worker_id, '')
		FROM job_executions e
		JOIN job_definitions d ON d.id = e.job_definition_id
		ORDER BY e.created_at DESC
		LIMIT 6`).Rows()
	if err != nil {
		return DashboardSummary{}, err
	}
	defer rows.Close()

	recent := make([]RecentExecution, 0, 6)
	for rows.Next() {
		var r RecentExecution
		if err := rows.Scan(&r.ID, &r.DefinitionName, &r.PluginKey, &r.Action, &r.Status, &r.WorkerID); err != nil {
			return DashboardSummary{}, err
		}
		recent = append(recent, r)
	}
	if err := rows.Err(); err != nil {
		return DashboardSummary{}, err
	}

	item := DashboardSummary{
		PluginCount:        len(pluginList),
		AccountTypeCount:   accountTypeCount,
		AccountCount:       accountCount,
		EmailAccountCount:  emailAccountCount,
		JobDefinitionCount: jobDefinitionCount,
		JobExecutionCount:  jobExecutionCount,
		TriggerCount:       triggerCount,
		AgentCount:         agentCount,
		RecentExecutions:   recent,
	}
	s.writeDashboardCache(ctx, item)
	return item, nil
}

func (s Service) readStatusCache(ctx context.Context) (Status, bool) {
	if s.rdb == nil {
		return Status{}, false
	}
	raw, err := s.rdb.Get(ctx, "system:status").Bytes()
	if err != nil {
		return Status{}, false
	}
	var item Status
	if err := json.Unmarshal(raw, &item); err != nil {
		return Status{}, false
	}
	return item, true
}

func (s Service) writeStatusCache(ctx context.Context, item Status) {
	s.writeCache(ctx, "system:status", item)
}

func (s Service) readDashboardCache(ctx context.Context) (DashboardSummary, bool) {
	if s.rdb == nil {
		return DashboardSummary{}, false
	}
	raw, err := s.rdb.Get(ctx, "system:dashboard").Bytes()
	if err != nil {
		return DashboardSummary{}, false
	}
	var item DashboardSummary
	if err := json.Unmarshal(raw, &item); err != nil {
		return DashboardSummary{}, false
	}
	return item, true
}

func (s Service) writeDashboardCache(ctx context.Context, item DashboardSummary) {
	s.writeCache(ctx, "system:dashboard", item)
}

func (s Service) writeCache(ctx context.Context, key string, item any) {
	if s.rdb == nil {
		return
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, key, raw, 3*time.Second).Err()
}
