package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/backend/internal/model"
	"octomanger/backend/internal/octomodule"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/worker/adapter"
	genericadapter "octomanger/backend/internal/worker/adapter/generic"
	"octomanger/backend/internal/worker/bridge"
)

const (
	JobStatusQueued   int16 = 0
	JobStatusRunning  int16 = 1
	JobStatusDone     int16 = 2
	JobStatusFailed   int16 = 3
	JobStatusCanceled int16 = 4
)

type JobExecutionResult struct {
	RunID        uint64
	AccountID    uint64
	Identifier   string
	Status       string
	Result       map[string]any
	ErrorCode    string
	ErrorMessage string
	Session      *adapter.Session
	StartedAt    time.Time
	EndedAt      *time.Time
}

type JobExecutionSummary struct {
	JobID             uint64
	Status            int16
	MatchedAccounts   int
	ProcessedAccounts int
	ErrorCode         string
	ErrorMessage      string
	Results           []JobExecutionResult
}

type JobExecutorOptions struct {
	Logger             *zap.Logger
	JobRepo            repository.JobRepository
	AccountRepo        repository.AccountRepository
	AccountTypeRepo    repository.AccountTypeRepository
	JobRunRepo         repository.JobRunRepository
	AccountSessionRepo repository.AccountSessionRepository
	PythonBridge       bridge.PythonBridge
	ModuleDir          string
	WorkerID           string
}

type jobExecutor struct {
	logger             *zap.Logger
	jobRepo            repository.JobRepository
	accountRepo        repository.AccountRepository
	accountTypeRepo    repository.AccountTypeRepository
	jobRunRepo         repository.JobRunRepository
	accountSessionRepo repository.AccountSessionRepository
	pythonBridge       bridge.PythonBridge
	moduleDir          string
	workerID           string

	mu              sync.Mutex
	genericAdapters map[string]adapter.Adapter
}

func NewJobExecutor(opts JobExecutorOptions) JobExecutor {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	workerID := strings.TrimSpace(opts.WorkerID)
	if workerID == "" {
		host, err := os.Hostname()
		if err != nil {
			host = "worker"
		}
		workerID = fmt.Sprintf("%s:%d", host, os.Getpid())
	}

	return &jobExecutor{
		logger:             logger,
		jobRepo:            opts.JobRepo,
		accountRepo:        opts.AccountRepo,
		accountTypeRepo:    opts.AccountTypeRepo,
		jobRunRepo:         opts.JobRunRepo,
		accountSessionRepo: opts.AccountSessionRepo,
		pythonBridge:       opts.PythonBridge,
		moduleDir:          strings.TrimSpace(opts.ModuleDir),
		workerID:           workerID,
		genericAdapters:    make(map[string]adapter.Adapter),
	}
}

func (e *jobExecutor) ExecuteJob(ctx context.Context, jobID uint64) (*JobExecutionSummary, error) {
	if jobID == 0 {
		return nil, invalidInput("job id is required")
	}

	job, err := e.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	summary := &JobExecutionSummary{
		JobID:   job.ID,
		Status:  job.Status,
		Results: []JobExecutionResult{},
	}

	if job.Status == JobStatusCanceled || job.Status == JobStatusDone || job.Status == JobStatusFailed {
		return summary, nil
	}

	if _, err := e.jobRepo.UpdateStatus(ctx, job.ID, JobStatusRunning); err != nil {
		return nil, err
	}
	summary.Status = JobStatusRunning

	accounts, err := e.accountRepo.ListByTypeKey(ctx, job.TypeKey)
	if err != nil {
		_, _ = e.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed)
		return nil, err
	}
	accounts = filterAccountsBySelector(accounts, job.Selector)
	summary.MatchedAccounts = len(accounts)
	if len(accounts) == 0 {
		if e.isJobCanceled(ctx, job.ID) {
			summary.Status = JobStatusCanceled
			return summary, nil
		}
		if _, err := e.jobRepo.UpdateStatus(ctx, job.ID, JobStatusDone); err != nil {
			return nil, err
		}
		summary.Status = JobStatusDone
		return summary, nil
	}

	moduleScript, err := e.resolveJobModule(ctx, job.TypeKey)
	if err != nil {
		summary.ErrorCode, summary.ErrorMessage = parseExecutionError(err)
		summary.Status = JobStatusFailed
		_, _ = e.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed)
		return summary, nil
	}

	for _, account := range accounts {
		if e.isJobCanceled(ctx, job.ID) {
			summary.Status = JobStatusCanceled
			return summary, nil
		}

		account = e.refreshAccountSpecIfNeeded(ctx, account)
		result := e.executeAction(ctx, job, account, moduleScript)
		summary.Results = append(summary.Results, result)
		summary.ProcessedAccounts++

		if result.Status != "success" {
			summary.Status = JobStatusFailed
			summary.ErrorCode = result.ErrorCode
			summary.ErrorMessage = result.ErrorMessage
			if _, err := e.jobRepo.UpdateStatus(ctx, job.ID, JobStatusFailed); err != nil {
				return nil, err
			}
			return summary, nil
		}
	}

	if e.isJobCanceled(ctx, job.ID) {
		summary.Status = JobStatusCanceled
		return summary, nil
	}
	if _, err := e.jobRepo.UpdateStatus(ctx, job.ID, JobStatusDone); err != nil {
		return nil, err
	}
	summary.Status = JobStatusDone
	return summary, nil
}

func (e *jobExecutor) resolveJobModule(ctx context.Context, typeKey string) (string, error) {
	accountType, err := e.accountTypeRepo.GetByKey(ctx, typeKey)
	if err != nil {
		return "", err
	}
	if !isGenericCategory(accountType.Category) {
		return "", invalidInput("job type must be a generic account type")
	}
	resolved, err := octomodule.ResolveEntryPath(e.moduleDir, accountType.Key, accountType.ScriptConfig)
	if err != nil {
		return "", err
	}
	if !octomodule.FileExists(resolved.EntryPath) {
		return "", fmt.Errorf("octoModule script does not exist: %s", resolved.EntryPath)
	}
	return resolved.EntryPath, nil
}

func (e *jobExecutor) executeAction(
	ctx context.Context,
	job *model.Job,
	account model.Account,
	moduleScript string,
) JobExecutionResult {
	startedAt := time.Now().UTC()
	item := JobExecutionResult{
		AccountID:  account.ID,
		Identifier: account.Identifier,
		Status:     "error",
		StartedAt:  startedAt,
	}

	selectedAdapter := e.pickAdapter(job.TypeKey)
	spec := decodeSpecMap(account.Spec)
	if err := selectedAdapter.ValidateSpec(spec); err != nil {
		execErr := fmt.Errorf("VALIDATION_FAILED: %w", err)
		errorCode, errorMessage := parseExecutionError(execErr)
		item.ErrorCode = errorCode
		item.ErrorMessage = errorMessage
		item.RunID, item.EndedAt = e.persistJobRun(ctx, job.ID, account.ID, startedAt, nil, execErr)
		return item
	}

	result, err := selectedAdapter.ExecuteAction(ctx, adapter.ActionRequest{
		RequestID:    fmt.Sprintf("%d:%d", job.ID, account.ID),
		TypeKey:      job.TypeKey,
		Action:       job.ActionKey,
		ModuleScript: moduleScript,
		Params:       decodeSpecMap(job.Params),
		Account: adapter.Account{
			ID:         fmt.Sprintf("%d", account.ID),
			Identifier: account.Identifier,
			Spec:       spec,
		},
	})
	if err != nil {
		errorCode, errorMessage := parseExecutionError(err)
		item.ErrorCode = errorCode
		item.ErrorMessage = errorMessage
		item.RunID, item.EndedAt = e.persistJobRun(ctx, job.ID, account.ID, startedAt, nil, err)
		return item
	}

	item.Status = result.Status
	item.Result = result.Result
	item.Session = result.Session
	item.RunID, item.EndedAt = e.persistJobRun(ctx, job.ID, account.ID, startedAt, &result, nil)
	if result.Session != nil {
		e.persistAccountSession(ctx, account.ID, result.Session)
	}
	return item
}

func (e *jobExecutor) pickAdapter(typeKey string) adapter.Adapter {
	key := strings.TrimSpace(typeKey)
	if key == "" {
		key = "generic"
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if selected, exists := e.genericAdapters[key]; exists {
		return selected
	}

	selected := genericadapter.New(key, e.pythonBridge)
	e.genericAdapters[key] = selected
	return selected
}

func (e *jobExecutor) persistJobRun(
	ctx context.Context,
	jobID uint64,
	accountID uint64,
	startedAt time.Time,
	result *adapter.Result,
	execErr error,
) (uint64, *time.Time) {
	endedAt := time.Now().UTC()

	var resultJSON []byte
	if result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			e.logger.Warn("failed to marshal job run result", zap.Uint64("job_id", jobID), zap.Error(err))
		} else {
			resultJSON = raw
		}
	}

	errorCode := ""
	errorMessage := ""
	if execErr != nil {
		errorCode, errorMessage = parseExecutionError(execErr)
	}

	accountIDValue := accountID
	jobRun := &model.JobRun{
		JobID:        jobID,
		AccountID:    &accountIDValue,
		WorkerID:     e.workerID,
		Attempt:      1,
		Result:       resultJSON,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		StartedAt:    startedAt,
		EndedAt:      &endedAt,
	}

	if err := e.jobRunRepo.Create(ctx, jobRun); err != nil {
		e.logger.Error("failed to persist job run", zap.Uint64("job_id", jobID), zap.Uint64("account_id", accountID), zap.Error(err))
	}
	return jobRun.ID, jobRun.EndedAt
}

func (e *jobExecutor) persistAccountSession(ctx context.Context, accountID uint64, session *adapter.Session) {
	if session == nil || accountID == 0 {
		return
	}

	payload, err := json.Marshal(session.Payload)
	if err != nil {
		e.logger.Warn("failed to marshal account session payload", zap.Uint64("account_id", accountID), zap.Error(err))
		return
	}

	item := &model.AccountSession{
		AccountID:        accountID,
		SessionType:      sessionType(session.Type),
		EncryptedPayload: payload,
		ExpiresAt:        parseSessionExpiry(session.ExpiresAt),
		State:            0,
	}
	if err := e.accountSessionRepo.Create(ctx, item); err != nil {
		e.logger.Error("failed to persist account session", zap.Uint64("account_id", accountID), zap.Error(err))
	}
}

func (e *jobExecutor) isJobCanceled(ctx context.Context, jobID uint64) bool {
	job, err := e.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			e.logger.Warn("failed to check job status", zap.Uint64("job_id", jobID), zap.Error(err))
		}
		return false
	}
	return job.Status == JobStatusCanceled
}

func (e *jobExecutor) refreshAccountSpecIfNeeded(ctx context.Context, account model.Account) model.Account {
	return tryRefreshAccountOAuth(ctx, account, func(id uint64, spec json.RawMessage) (*model.Account, error) {
		current := account
		current.ID = id
		current.Spec = spec
		if err := e.accountRepo.Update(ctx, &current); err != nil {
			return nil, err
		}
		return &current, nil
	})
}

func parseExecutionError(err error) (string, string) {
	if err == nil {
		return "", ""
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "EXECUTION_FAILED", "execution failed"
	}

	parts := strings.SplitN(message, ": ", 2)
	if len(parts) == 2 {
		code := strings.TrimSpace(parts[0])
		if isUpperCode(code) {
			return code, strings.TrimSpace(parts[1])
		}
	}
	return "EXECUTION_FAILED", message
}

func isUpperCode(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r == '_':
			continue
		case unicode.IsDigit(r):
			continue
		case unicode.IsUpper(r):
			continue
		default:
			return false
		}
	}
	return true
}

func sessionType(value string) int16 {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "token":
		return 1
	case "cookie":
		return 2
	default:
		return 0
	}
}

func parseSessionExpiry(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		normalized := parsed.UTC()
		return &normalized
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		normalized := parsed.UTC()
		return &normalized
	}
	return nil
}

func filterAccountsBySelector(accounts []model.Account, selectorRaw json.RawMessage) []model.Account {
	selector := decodeSpecMap(selectorRaw)
	if len(selector) == 0 {
		return accounts
	}

	accountIDSet := toUint64Set(selector["account_ids"])
	identifierSet := toStringSet(selector["identifiers"])

	identifierContains := ""
	if raw, ok := selector["identifier_contains"].(string); ok {
		identifierContains = strings.ToLower(strings.TrimSpace(raw))
	}

	filtered := make([]model.Account, 0, len(accounts))
	for _, account := range accounts {
		if len(accountIDSet) > 0 && !accountIDSet[account.ID] {
			continue
		}
		if len(identifierSet) > 0 && !identifierSet[account.Identifier] {
			continue
		}
		if identifierContains != "" && !strings.Contains(strings.ToLower(account.Identifier), identifierContains) {
			continue
		}
		filtered = append(filtered, account)
	}

	if rawLimit, ok := selector["limit"].(float64); ok {
		limit := int(rawLimit)
		if limit > 0 && len(filtered) > limit {
			return filtered[:limit]
		}
	}
	return filtered
}

func toStringSet(value any) map[string]bool {
	if value == nil {
		return nil
	}

	result := map[string]bool{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result[text] = true
			}
		}
	case []string:
		for _, item := range typed {
			result[item] = true
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func toUint64Set(value any) map[uint64]bool {
	if value == nil {
		return nil
	}

	result := map[uint64]bool{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			switch v := item.(type) {
			case float64:
				if v > 0 {
					result[uint64(v)] = true
				}
			case string:
				if parsed, err := parseUint64(v); err == nil && parsed > 0 {
					result[parsed] = true
				}
			}
		}
	case []uint64:
		for _, item := range typed {
			result[item] = true
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func parseUint64(value string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(value), 10, 64)
}
