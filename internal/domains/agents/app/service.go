package agentapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	agentdomain "octomanger/internal/domains/agents/domain"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	plugindomain "octomanger/internal/domains/plugins/domain"
	"octomanger/internal/platform/logbatch"
)

// pluginExecutor is the narrow interface agentapp needs from the plugin backend.
type pluginExecutor interface {
	Execute(ctx context.Context, pluginKey string, request plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error
}

const statusCacheTTL = 5 * time.Second

type Service struct {
	logger            *zap.Logger
	repo              agentpostgres.Repository
	plugins           pluginExecutor
	rdb               *redis.Client // nil = cache disabled
	workerID          string
	loopInterval      time.Duration
	errorBackoff      time.Duration
	mu                sync.Mutex
	activeAgentCancel map[int64]context.CancelFunc
}

func New(
	logger *zap.Logger,
	repo agentpostgres.Repository,
	plugins pluginExecutor,
	rdb *redis.Client,
	workerID string,
	loopInterval time.Duration,
	errorBackoff time.Duration,
) *Service {
	return &Service{
		logger:            logger,
		repo:              repo,
		plugins:           plugins,
		rdb:               rdb,
		workerID:          workerID,
		loopInterval:      loopInterval,
		errorBackoff:      errorBackoff,
		activeAgentCancel: map[int64]context.CancelFunc{},
	}
}

// ── Cache helpers ────────────────────────────────────────────────────────────

func statusCacheKey(id int64) string {
	return fmt.Sprintf("agent:status:%d", id)
}

func (s *Service) writeStatusCache(ctx context.Context, status agentdomain.AgentStatus) {
	if s.rdb == nil {
		return
	}
	data, err := json.Marshal(status)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, statusCacheKey(status.ID), data, statusCacheTTL).Err()
}

func (s *Service) invalidateStatusCache(ctx context.Context, id int64) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Del(ctx, statusCacheKey(id)).Err()
}

func (s *Service) readStatusCache(ctx context.Context, id int64) (*agentdomain.AgentStatus, bool) {
	if s.rdb == nil {
		return nil, false
	}
	data, err := s.rdb.Get(ctx, statusCacheKey(id)).Bytes()
	if err != nil {
		return nil, false
	}
	var status agentdomain.AgentStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, false
	}
	return &status, true
}

// ── Public API ───────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context) ([]agentdomain.Agent, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, input agentdomain.CreateAgentInput) (*agentdomain.Agent, error) {
	return s.repo.Create(ctx, input)
}

func (s *Service) Patch(ctx context.Context, agentID int64, input agentdomain.PatchAgentInput) (*agentdomain.Agent, error) {
	return s.repo.Patch(ctx, agentID, input)
}

func (s *Service) Start(ctx context.Context, agentID int64) error {
	if err := s.repo.SetDesiredState(ctx, agentID, "running"); err != nil {
		return err
	}
	s.invalidateStatusCache(ctx, agentID)
	return nil
}

func (s *Service) Stop(ctx context.Context, agentID int64) error {
	if err := s.repo.SetDesiredState(ctx, agentID, "stopped"); err != nil {
		return err
	}
	s.invalidateStatusCache(ctx, agentID)

	s.mu.Lock()
	cancel := s.activeAgentCancel[agentID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (s *Service) Get(ctx context.Context, agentID int64) (*agentdomain.Agent, error) {
	return s.repo.Get(ctx, agentID)
}

// GetStatus returns the lightweight status snapshot for an agent.
// Redis cache is tried first (TTL 5s); on miss the DB is queried and the result cached.
func (s *Service) GetStatus(ctx context.Context, agentID int64) (*agentdomain.AgentStatus, error) {
	if status, ok := s.readStatusCache(ctx, agentID); ok {
		return status, nil
	}
	agent, err := s.repo.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}
	status := agentToStatus(agent)
	s.writeStatusCache(ctx, status)
	return &status, nil
}

func (s *Service) Delete(ctx context.Context, agentID int64) error {
	s.invalidateStatusCache(ctx, agentID)
	s.mu.Lock()
	cancel := s.activeAgentCancel[agentID]
	delete(s.activeAgentCancel, agentID)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return s.repo.Delete(ctx, agentID)
}

func (s *Service) ListLogsAfter(ctx context.Context, agentID int64, afterID int64) ([]agentdomain.AgentLog, error) {
	return s.repo.ListLogsAfter(ctx, agentID, afterID)
}

// ── Supervisor ───────────────────────────────────────────────────────────────

func (s *Service) RunSupervisor(ctx context.Context, scanInterval time.Duration) error {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		if err := s.syncAgents(ctx); err != nil {
			s.logger.Sugar().Errorw("agent supervisor sync failed", "error", err)
		}

		select {
		case <-ctx.Done():
			s.stopAll()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) syncAgents(ctx context.Context) error {
	items, err := s.repo.ListDesiredRunning(ctx)
	if err != nil {
		return err
	}

	desired := make(map[int64]agentdomain.Agent, len(items))
	for _, item := range items {
		desired[item.ID] = item
	}

	s.mu.Lock()
	for id, cancel := range s.activeAgentCancel {
		if _, exists := desired[id]; exists {
			continue
		}
		cancel()
		delete(s.activeAgentCancel, id)
	}
	for _, item := range items {
		if _, exists := s.activeAgentCancel[item.ID]; exists {
			continue
		}
		agentCtx, cancel := context.WithCancel(ctx)
		s.activeAgentCancel[item.ID] = cancel
		go s.runAgentLoop(agentCtx, item)
	}
	s.mu.Unlock()

	return nil
}

func (s *Service) runAgentLoop(ctx context.Context, agent agentdomain.Agent) {
	defer func() {
		_ = s.updateRuntimeState(context.Background(), agent.ID, "stopped", "", nil)

		s.mu.Lock()
		delete(s.activeAgentCancel, agent.ID)
		s.mu.Unlock()
	}()

	for {
		heartbeat := time.Now().UTC()
		if err := s.updateRuntimeState(ctx, agent.ID, "running", "", &heartbeat); err != nil {
			s.logger.Sugar().Errorw("update agent runtime state failed", "agent_id", agent.ID, "error", err)
		}

		var pluginError string

		batchCtx, cancelBatch := context.WithCancel(context.Background())
		batcher := logbatch.New[agentdomain.AgentLogEntry](func(ctx context.Context, entries []agentdomain.AgentLogEntry) error {
			return s.repo.AppendLogBatch(ctx, entries)
		})
		go batcher.Run(batchCtx)

		runErr := s.plugins.Execute(ctx, agent.PluginKey, plugindomain.ExecutionRequest{
			Mode:   "agent",
			Action: agent.Action,
			Input:  agent.Input,
			Context: map[string]any{
				"agent_id":   agent.ID,
				"worker_id":  s.workerID,
				"started_at": heartbeat.Format(time.RFC3339),
			},
		}, func(event plugindomain.ExecutionEvent) {
			if event.Type == "" && event.Message == "" {
				return // skip empty heartbeat events
			}
			if event.Type == "error" && pluginError == "" {
				pluginError = event.Message
				if pluginError == "" {
					pluginError = event.Error
				}
			}
			batcher.Add(agentdomain.AgentLogEntry{
				AgentID:   agent.ID,
				EventType: event.Type,
				Message:   event.Message,
				Payload:   event.Data,
			})
		})

		cancelBatch()
		batcher.Wait()
		if runErr == nil && pluginError != "" {
			runErr = errors.New(pluginError)
		}

		heartbeat = time.Now().UTC()
		if runErr != nil {
			_ = s.repo.AppendLog(ctx, agent.ID, "error", runErr.Error(), nil)
			_ = s.updateRuntimeState(ctx, agent.ID, "error", runErr.Error(), &heartbeat)
			backoffTimer := time.NewTimer(s.errorBackoff)
			select {
			case <-ctx.Done():
				backoffTimer.Stop()
				return
			case <-backoffTimer.C:
				continue
			}
		}

		_ = s.updateRuntimeState(ctx, agent.ID, "idle", "", &heartbeat)

		loopTimer := time.NewTimer(s.loopInterval)
		select {
		case <-ctx.Done():
			loopTimer.Stop()
			return
		case <-loopTimer.C:
		}
	}
}

// updateRuntimeState writes to DB then refreshes the Redis status cache.
func (s *Service) updateRuntimeState(ctx context.Context, agentID int64, runtimeState, lastError string, heartbeat *time.Time) error {
	if err := s.repo.UpdateRuntimeState(ctx, agentID, runtimeState, lastError, heartbeat); err != nil {
		return err
	}
	// Read back the full row (desired_state is only in DB) and cache it.
	if agent, err := s.repo.Get(ctx, agentID); err == nil {
		s.writeStatusCache(ctx, agentToStatus(agent))
	}
	return nil
}

func (s *Service) stopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, cancel := range s.activeAgentCancel {
		cancel()
		delete(s.activeAgentCancel, id)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func agentToStatus(a *agentdomain.Agent) agentdomain.AgentStatus {
	return agentdomain.AgentStatus{
		ID:              a.ID,
		RuntimeState:    a.RuntimeState,
		DesiredState:    a.DesiredState,
		LastError:       a.LastError,
		LastHeartbeatAt: a.LastHeartbeatAt,
		UpdatedAt:       a.UpdatedAt,
	}
}
