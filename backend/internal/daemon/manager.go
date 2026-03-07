// Package daemon manages long-lived Python processes for OctoModules that declare
// a "daemon" capability. Each account gets its own persistent subprocess which runs
// an initialization phase followed by a continuous event loop.
//
// Convention: an AccountType opts into daemon mode by adding a "daemon" key to its
// capabilities JSON:
//
//	{
//	  "actions": [...],
//	  "daemon": { "action": "WATCH" }
//	}
//
// The Python module for daemon actions must:
//  1. Read the standard JSON input from stdin (same as regular modules)
//  2. Print one NDJSON line per event to stdout (flush=True after each)
//  3. Use {"status": "init_ok"} after initialization is complete
//  4. Use {"status": "event", "result": {...}} for each subsequent event
//  5. Use {"status": "done"} to signal a clean, intentional exit
//  6. Use {"status": "error", "error_code": "...", "error_message": "..."} on fatal errors
//
// The Go daemon manager stores each received event as a job_run record linked to a
// sentinel job created at startup. This makes events visible in the existing UI.
package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/backend/internal/model"
	"octomanger/backend/internal/octomodule"
)

// Config holds daemon manager settings.
type Config struct {
	PythonBin  string
	ModuleDir  string
	WorkerID   string        // written to job_run.worker_id; defaults to "daemon"
	RestartMin time.Duration // minimum backoff before restarting a crashed process
	RestartMax time.Duration // maximum backoff ceiling
}

func (c *Config) setDefaults() {
	if c.WorkerID == "" {
		c.WorkerID = "daemon"
	}
	if c.RestartMin <= 0 {
		c.RestartMin = 5 * time.Second
	}
	if c.RestartMax <= 0 {
		c.RestartMax = 5 * time.Minute
	}
}

// Manager discovers daemon-enabled account types and manages their Python processes.
type Manager struct {
	db     *gorm.DB
	cfg    Config
	logger *zap.Logger
}

func NewManager(db *gorm.DB, cfg Config, logger *zap.Logger) *Manager {
	cfg.setDefaults()
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Manager{db: db, cfg: cfg, logger: logger}
}

// Run blocks until ctx is cancelled. It discovers daemon-enabled account types, resolves
// their script paths, and starts a persistent subprocess per active account.
func (m *Manager) Run(ctx context.Context) error {
	types, err := m.loadDaemonTypes(ctx)
	if err != nil {
		return err
	}
	if len(types) == 0 {
		m.logger.Info("daemon: no daemon-enabled account types found; waiting for context cancellation")
		<-ctx.Done()
		return nil
	}

	var wg sync.WaitGroup
	for _, entry := range types {
		entry := entry

		accounts, err := m.loadAccounts(ctx, entry.typeKey)
		if err != nil {
			m.logger.Warn("daemon: failed to load accounts", zap.String("type_key", entry.typeKey), zap.Error(err))
			continue
		}
		if len(accounts) == 0 {
			m.logger.Info("daemon: no active accounts", zap.String("type_key", entry.typeKey))
			continue
		}

		sentinelJobID, err := m.ensureSentinelJob(ctx, entry.typeKey, entry.action)
		if err != nil {
			m.logger.Warn("daemon: failed to create sentinel job", zap.String("type_key", entry.typeKey), zap.Error(err))
			continue
		}

		for _, acc := range accounts {
			acc := acc
			wg.Add(1)
			go func() {
				defer wg.Done()
				m.runProcessLoop(ctx, entry.scriptPath, entry.action, sentinelJobID, acc)
			}()
		}
	}

	wg.Wait()
	return nil
}

// daemonEntry holds resolved info for one daemon-enabled account type.
type daemonEntry struct {
	typeKey    string
	action     string
	scriptPath string
}

func (m *Manager) loadDaemonTypes(ctx context.Context) ([]daemonEntry, error) {
	var types []model.AccountType
	err := m.db.WithContext(ctx).
		Where("category = 'generic' AND capabilities->>'daemon' IS NOT NULL").
		Find(&types).Error
	if err != nil {
		return nil, fmt.Errorf("daemon: query account types: %w", err)
	}

	var entries []daemonEntry
	for _, at := range types {
		action, ok := parseDaemonAction(at.Capabilities)
		if !ok {
			continue
		}

		resolved, err := octomodule.ResolveEntryPath(m.cfg.ModuleDir, at.Key, at.ScriptConfig)
		if err != nil {
			m.logger.Warn("daemon: cannot resolve script path", zap.String("type_key", at.Key), zap.Error(err))
			continue
		}
		if !octomodule.FileExists(resolved.EntryPath) {
			m.logger.Warn("daemon: script file not found", zap.String("type_key", at.Key), zap.String("path", resolved.EntryPath))
			continue
		}

		entries = append(entries, daemonEntry{
			typeKey:    at.Key,
			action:     action,
			scriptPath: resolved.EntryPath,
		})
		m.logger.Info("daemon: registered",
			zap.String("type_key", at.Key),
			zap.String("action", action),
			zap.String("script", resolved.EntryPath),
		)
	}
	return entries, nil
}

func parseDaemonAction(capabilities json.RawMessage) (string, bool) {
	if len(capabilities) == 0 {
		return "", false
	}
	var caps struct {
		Daemon *struct {
			Action string `json:"action"`
		} `json:"daemon"`
	}
	if err := json.Unmarshal(capabilities, &caps); err != nil || caps.Daemon == nil {
		return "", false
	}
	action := caps.Daemon.Action
	if action == "" {
		action = "WATCH"
	}
	return action, true
}

func (m *Manager) loadAccounts(ctx context.Context, typeKey string) ([]model.Account, error) {
	var accounts []model.Account
	err := m.db.WithContext(ctx).
		Where("type_key = ? AND status = 1", typeKey). // status 1 = active
		Find(&accounts).Error
	return accounts, err
}

// ensureSentinelJob finds or creates a long-running "daemon" job that serves as the
// parent for all job_run event records produced by this daemon type.
func (m *Manager) ensureSentinelJob(ctx context.Context, typeKey, action string) (uint64, error) {
	actionKey := "DAEMON:" + action
	var job model.Job
	err := m.db.WithContext(ctx).
		Where("type_key = ? AND action_key = ? AND status = ?", typeKey, actionKey, int16(1)). // 1 = running
		First(&job).Error
	if err == nil {
		return job.ID, nil
	}

	job = model.Job{
		TypeKey:   typeKey,
		ActionKey: actionKey,
		Selector:  json.RawMessage("{}"),
		Params:    json.RawMessage("{}"),
		Status:    int16(1), // running
	}
	if err := m.db.WithContext(ctx).Create(&job).Error; err != nil {
		return 0, fmt.Errorf("create sentinel job: %w", err)
	}
	return job.ID, nil
}

// runProcessLoop restarts the Python process for a single account whenever it exits,
// applying exponential backoff on failures.
func (m *Manager) runProcessLoop(ctx context.Context, scriptPath, action string, jobID uint64, acc model.Account) {
	log := m.logger.With(
		zap.String("account", acc.Identifier),
		zap.String("type_key", acc.TypeKey),
	)

	backoff := m.cfg.RestartMin
	for {
		if ctx.Err() != nil {
			return
		}

		log.Info("daemon: starting process")
		err := m.runProcess(ctx, scriptPath, action, jobID, acc)

		if ctx.Err() != nil {
			return // clean shutdown, no restart
		}

		if err != nil {
			log.Warn("daemon: process exited with error", zap.Error(err), zap.Duration("restart_in", backoff))
		} else {
			log.Info("daemon: process exited cleanly, scheduling restart")
			backoff = m.cfg.RestartMin // reset on clean exits
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > m.cfg.RestartMax {
			backoff = m.cfg.RestartMax
		}
	}
}

// runProcess starts a single Python subprocess, reads its NDJSON stdout line by line,
// and persists events. It returns when the process exits.
func (m *Manager) runProcess(ctx context.Context, scriptPath, action string, jobID uint64, acc model.Account) error {
	var spec map[string]any
	if len(acc.Spec) > 0 {
		_ = json.Unmarshal(acc.Spec, &spec)
	}

	input := map[string]any{
		"action": action,
		"account": map[string]any{
			"identifier": acc.Identifier,
			"spec":       spec,
		},
		"params":  map[string]any{},
		"context": map[string]any{"request_id": fmt.Sprintf("daemon:%d", acc.ID)},
	}
	rawInput, err := json.Marshal(input)
	if err != nil {
		return err
	}

	binary := m.cfg.PythonBin
	if venv := resolveVenvPython(filepath.Dir(scriptPath)); venv != "" {
		binary = venv
	}

	cmd := exec.CommandContext(ctx, binary, scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	cmd.Env = []string{"PYTHONUNBUFFERED=1", "PYTHONIOENCODING=UTF-8"}
	cmd.Stdin = bytes.NewReader(rawInput)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	log := m.logger.With(
		zap.String("account", acc.Identifier),
		zap.Int("pid", cmd.Process.Pid),
	)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var evt struct {
			Status       string         `json:"status"`
			Result       map[string]any `json:"result,omitempty"`
			ErrorCode    string         `json:"error_code,omitempty"`
			ErrorMessage string         `json:"error_message,omitempty"`
		}
		if err := json.Unmarshal(line, &evt); err != nil {
			log.Warn("daemon: unreadable output line", zap.String("raw", string(line)))
			continue
		}

		switch evt.Status {
		case "init_ok":
			log.Info("daemon: initialized")
		case "event":
			log.Info("daemon: event received", zap.Any("result", evt.Result))
			m.persistEvent(jobID, acc.ID, evt.Result)
		case "done":
			log.Info("daemon: process signaled done")
		case "error":
			log.Warn("daemon: module error",
				zap.String("code", evt.ErrorCode),
				zap.String("message", evt.ErrorMessage),
			)
			m.persistFailedEvent(jobID, acc.ID, evt.ErrorCode, evt.ErrorMessage)
		default:
			log.Warn("daemon: unknown status", zap.String("status", evt.Status))
		}
	}

	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("process exited: %w (stderr=%s)", err, stderrBuf.String())
	}
	return nil
}

func (m *Manager) persistEvent(jobID, accountID uint64, result map[string]any) {
	resultBytes, _ := json.Marshal(result)
	now := time.Now()
	run := model.JobRun{
		JobID:     jobID,
		AccountID: &accountID,
		WorkerID:  m.cfg.WorkerID,
		Attempt:   1,
		Result:    resultBytes,
		StartedAt: now,
		EndedAt:   &now,
	}
	if err := m.db.Create(&run).Error; err != nil {
		m.logger.Warn("daemon: failed to persist event", zap.Error(err))
	}
}

func (m *Manager) persistFailedEvent(jobID, accountID uint64, code, message string) {
	now := time.Now()
	run := model.JobRun{
		JobID:        jobID,
		AccountID:    &accountID,
		WorkerID:     m.cfg.WorkerID,
		Attempt:      1,
		ErrorCode:    code,
		ErrorMessage: message,
		StartedAt:    now,
		EndedAt:      &now,
	}
	if err := m.db.Create(&run).Error; err != nil {
		m.logger.Warn("daemon: failed to persist error event", zap.Error(err))
	}
}

func resolveVenvPython(dir string) string {
	var candidate string
	if runtime.GOOS == "windows" {
		candidate = filepath.Join(dir, ".venv", "Scripts", "python.exe")
	} else {
		candidate = filepath.Join(dir, ".venv", "bin", "python")
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}
