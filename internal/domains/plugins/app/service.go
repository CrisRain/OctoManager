package pluginapp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	plugindomain "octomanger/internal/domains/plugins/domain"
)

const scannerMaxTokenSize = 4 * 1024 * 1024 // 4MB

type Repository interface {
	List(ctx context.Context) ([]plugindomain.Plugin, error)
	Get(ctx context.Context, key string) (*plugindomain.Plugin, error)
}

type SettingsStore interface {
	GetConfig(ctx context.Context, key string) (json.RawMessage, error)
}

type ExecutionTimeouts struct {
	Account time.Duration
	Job     time.Duration
	Agent   time.Duration
}

type Service struct {
	repo              Repository
	pythonBin         string
	sdkDir            string
	settingsStore     SettingsStore
	executionTimeouts ExecutionTimeouts
}

func New(repo Repository, pythonBin string, sdkDir string) Service {
	return Service{
		repo:              repo,
		pythonBin:         strings.TrimSpace(pythonBin),
		sdkDir:            strings.TrimSpace(sdkDir),
		executionTimeouts: defaultExecutionTimeouts(),
	}
}

func (s Service) WithSettingsStore(store SettingsStore) Service {
	s.settingsStore = store
	return s
}

func (s Service) WithExecutionTimeouts(timeouts ExecutionTimeouts) Service {
	s.executionTimeouts = normalizeExecutionTimeouts(timeouts)
	return s
}

func (s Service) List(ctx context.Context) ([]plugindomain.Plugin, error) {
	return s.repo.List(ctx)
}

// AccountTypeSpec is the subset of account_type.{key}.json that maps to an AccountType record.
type AccountTypeSpec struct {
	Key          string         `json:"key"`
	Name         string         `json:"name"`
	Category     string         `json:"category"`
	Schema       map[string]any `json:"schema"`
	Capabilities map[string]any `json:"capabilities"`
}

// SyncAccountTypeFunc is called once per plugin with the parsed account type spec.
// The caller (Bootstrap) wires this to accountTypeService.Upsert.
type SyncAccountTypeFunc func(ctx context.Context, spec AccountTypeSpec) error

// SyncAccountTypes reads every plugin's account_type.{key}.json and calls fn for each one.
// Missing files are silently skipped. Parse errors are returned immediately.
func (s Service) SyncAccountTypes(ctx context.Context, fn SyncAccountTypeFunc) error {
	plugins, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list plugins for sync: %w", err)
	}

	for _, plugin := range plugins {
		filename := filepath.Join(plugin.Directory, fmt.Sprintf("account_type.%s.json", plugin.Manifest.Key))
		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue // no account type file for this plugin — skip
			}
			return fmt.Errorf("read account type file %s: %w", filename, err)
		}

		var spec AccountTypeSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("parse account type file %s: %w", filename, err)
		}
		if spec.Key == "" {
			spec.Key = plugin.Manifest.Key
		}
		if spec.Name == "" {
			spec.Name = plugin.Manifest.Name
		}
		if spec.Category == "" {
			spec.Category = "generic"
		}

		if err := fn(ctx, spec); err != nil {
			return fmt.Errorf("sync account type %s: %w", spec.Key, err)
		}
	}
	return nil
}

func (s Service) Get(ctx context.Context, key string) (*plugindomain.Plugin, error) {
	return s.repo.Get(ctx, key)
}

func (s Service) Execute(
	ctx context.Context,
	pluginKey string,
	request plugindomain.ExecutionRequest,
	onEvent func(plugindomain.ExecutionEvent),
) error {
	plugin, err := s.repo.Get(ctx, pluginKey)
	if err != nil {
		return err
	}

	request, err = s.injectSettings(ctx, pluginKey, request)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal plugin request: %w", err)
	}

	execCtx := ctx
	timeout := s.executionTimeoutForRequest(request)
	cancel := func() {}
	if timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	entrypoint := plugin.Manifest.Entrypoint
	command := exec.CommandContext(execCtx, s.pythonBinary(), entrypoint)
	command.Dir = plugin.Directory
	command.Env = append(command.Environ(), fmt.Sprintf("PYTHONPATH=%s", s.pythonPath()))

	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open plugin stdout: %w", err)
	}
	command.Stderr = command.Stdout
	command.Stdin = strings.NewReader(string(payload))

	if err := command.Start(); err != nil {
		return fmt.Errorf("start plugin command: %w", err)
	}

	var receivedError bool
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxTokenSize)
	var warnedDeprecatedStatus bool
	for scanner.Scan() {
		line := scanner.Bytes()

		event, usesDeprecatedStatus := decodeExecutionEventLine(line)
		if usesDeprecatedStatus && !warnedDeprecatedStatus {
			warnedDeprecatedStatus = true
			fmt.Fprintf(os.Stderr, "WARN: plugin %q emitted deprecated status-based events; migrate to type-based event protocol\n", pluginKey)
		}

		if event.Type == "error" {
			receivedError = true
		}
		if onEvent != nil {
			onEvent(event)
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("read plugin output: line exceeds %d bytes", scannerMaxTokenSize)
		}
		return fmt.Errorf("read plugin output: %w", err)
	}

	if err := command.Wait(); err != nil {
		if timeout > 0 && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			message := fmt.Sprintf("plugin execution timed out after %s", timeout)
			if onEvent != nil {
				onEvent(plugindomain.ExecutionEvent{Type: "error", Message: message, Error: "TIMEOUT"})
			}
			return errors.New(message)
		}
		if receivedError {
			// The plugin already communicated the failure via an error event; suppress the
			// redundant non-zero exit-code error so callers don't see it twice.
			return nil
		}
		// Process failed without emitting an error event — surface it as one.
		if onEvent != nil {
			onEvent(plugindomain.ExecutionEvent{Type: "error", Message: err.Error(), Error: err.Error()})
		}
		return fmt.Errorf("wait plugin command: %w", err)
	}

	return nil
}

func (s Service) injectSettings(ctx context.Context, pluginKey string, request plugindomain.ExecutionRequest) (plugindomain.ExecutionRequest, error) {
	settings, err := s.loadSettings(ctx, pluginKey)
	if err != nil {
		return request, err
	}

	if request.Context == nil {
		request.Context = map[string]any{}
	}
	request.Context["settings"] = settings
	return request, nil
}

func (s Service) loadSettings(ctx context.Context, pluginKey string) (map[string]any, error) {
	if s.settingsStore == nil || strings.TrimSpace(pluginKey) == "" {
		return map[string]any{}, nil
	}

	raw, err := s.settingsStore.GetConfig(ctx, settingsKey(pluginKey))
	if err != nil {
		return nil, fmt.Errorf("get plugin settings %s: %w", pluginKey, err)
	}
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return map[string]any{}, nil
	}

	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("decode plugin settings %s: %w", pluginKey, err)
	}
	if settings == nil {
		return map[string]any{}, nil
	}
	return settings, nil
}

func (s Service) executionTimeoutForRequest(request plugindomain.ExecutionRequest) time.Duration {
	if source := strings.TrimSpace(asString(request.Context["source"])); source == "account-execute" {
		return s.executionTimeouts.Account
	}

	switch strings.ToLower(strings.TrimSpace(request.Mode)) {
	case "agent":
		return s.executionTimeouts.Agent
	case "job":
		return s.executionTimeouts.Job
	case "account":
		return s.executionTimeouts.Account
	default:
		return 0
	}
}

func settingsKey(pluginKey string) string {
	return "plugin_settings:" + pluginKey
}

func defaultExecutionTimeouts() ExecutionTimeouts {
	return ExecutionTimeouts{
		Account: 60 * time.Second,
		Job:     10 * time.Minute,
		Agent:   0,
	}
}

func normalizeExecutionTimeouts(in ExecutionTimeouts) ExecutionTimeouts {
	if in.Account < 0 {
		in.Account = 0
	}
	if in.Job < 0 {
		in.Job = 0
	}
	if in.Agent < 0 {
		in.Agent = 0
	}
	return in
}

func decodeExecutionEventLine(line []byte) (plugindomain.ExecutionEvent, bool) {
	raw := map[string]any{}
	if err := json.Unmarshal(line, &raw); err != nil {
		return plugindomain.ExecutionEvent{Type: "log", Message: string(line)}, false
	}

	if eventType := strings.TrimSpace(asString(raw["type"])); eventType != "" {
		event := plugindomain.ExecutionEvent{
			Type:     eventType,
			Message:  strings.TrimSpace(asString(raw["message"])),
			Progress: asInt(raw["progress"]),
			Data:     asMap(raw["data"]),
			Error:    strings.TrimSpace(asString(raw["error"])),
		}
		if event.Type == "error" && event.Message == "" {
			event.Message = event.Error
		}
		return event, false
	}

	status := strings.ToLower(strings.TrimSpace(asString(raw["status"])))
	if status == "" {
		return plugindomain.ExecutionEvent{Type: "log", Message: string(line)}, false
	}

	switch status {
	case "log":
		message := strings.TrimSpace(asString(raw["message"]))
		if message == "" {
			message = strings.TrimSpace(asString(raw["detail_message"]))
		}
		if message == "" {
			message = string(line)
		}
		return plugindomain.ExecutionEvent{Type: "log", Message: message}, true
	case "success":
		data := asMap(raw["result"])
		if data == nil {
			data = map[string]any{}
			if value, exists := raw["result"]; exists && value != nil {
				data["value"] = value
			}
		}
		return plugindomain.ExecutionEvent{
			Type:    "result",
			Message: strings.TrimSpace(asString(raw["message"])),
			Data:    data,
		}, true
	case "error":
		errorCode := strings.TrimSpace(asString(raw["error_code"]))
		message := strings.TrimSpace(asString(raw["error_message"]))
		if message == "" {
			message = strings.TrimSpace(asString(raw["message"]))
		}
		if message == "" {
			message = errorCode
		}
		return plugindomain.ExecutionEvent{
			Type:    "error",
			Message: message,
			Error:   errorCode,
			Data:    asMap(raw["result"]),
		}, true
	case "event", "init_ok", "done":
		data := asMap(raw["result"])
		if data == nil {
			data = map[string]any{}
		}
		data["status"] = status
		return plugindomain.ExecutionEvent{
			Type:    "progress",
			Message: strings.TrimSpace(asString(raw["message"])),
			Data:    data,
		}, true
	default:
		return plugindomain.ExecutionEvent{Type: "log", Message: string(line)}, true
	}
}

func asString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func asMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func asInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func (s Service) pythonBinary() string {
	if s.pythonBin != "" {
		return s.pythonBin
	}
	return "python3"
}

func (s Service) pythonPath() string {
	if s.sdkDir == "" {
		return ""
	}
	abs, err := filepath.Abs(s.sdkDir)
	if err != nil {
		return s.sdkDir
	}
	// Preserve any existing PYTHONPATH so system packages remain accessible.
	if existing := os.Getenv("PYTHONPATH"); existing != "" {
		return abs + string(os.PathListSeparator) + existing
	}
	return abs
}
