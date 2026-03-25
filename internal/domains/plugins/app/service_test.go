package pluginapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	plugindomain "octomanger/internal/domains/plugins/domain"
)

func TestDecodeExecutionEventLine(t *testing.T) {
	t.Parallel()

	t.Run("type-protocol", func(t *testing.T) {
		event, deprecated := decodeExecutionEventLine([]byte(`{"type":"result","message":"ok","data":{"value":1}}`))
		if deprecated {
			t.Fatalf("expected type protocol not to be marked deprecated")
		}
		if event.Type != "result" {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if event.Message != "ok" {
			t.Fatalf("unexpected message: %s", event.Message)
		}
		if got := event.Data["value"]; got != float64(1) {
			t.Fatalf("unexpected data value: %#v", got)
		}
	})

	t.Run("status-success", func(t *testing.T) {
		event, deprecated := decodeExecutionEventLine([]byte(`{"status":"success","result":{"answer":42}}`))
		if !deprecated {
			t.Fatalf("expected status protocol to be marked deprecated")
		}
		if event.Type != "result" {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if got := event.Data["answer"]; got != float64(42) {
			t.Fatalf("unexpected result data: %#v", got)
		}
	})

	t.Run("status-error", func(t *testing.T) {
		event, deprecated := decodeExecutionEventLine([]byte(`{"status":"error","error_code":"NETWORK_ERROR","error_message":"offline"}`))
		if !deprecated {
			t.Fatalf("expected status protocol to be marked deprecated")
		}
		if event.Type != "error" {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if event.Error != "NETWORK_ERROR" {
			t.Fatalf("unexpected error code: %s", event.Error)
		}
		if event.Message != "offline" {
			t.Fatalf("unexpected message: %s", event.Message)
		}
	})

	t.Run("status-unknown-fallback-log", func(t *testing.T) {
		line := []byte(`{"status":"mystery","foo":"bar"}`)
		event, deprecated := decodeExecutionEventLine(line)
		if !deprecated {
			t.Fatalf("expected status protocol to be marked deprecated")
		}
		if event.Type != "log" {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if event.Message != string(line) {
			t.Fatalf("unexpected log message: %s", event.Message)
		}
	})

	t.Run("non-json-fallback-log", func(t *testing.T) {
		line := []byte("plain log line")
		event, deprecated := decodeExecutionEventLine(line)
		if deprecated {
			t.Fatalf("non-json output must not be marked deprecated")
		}
		if event.Type != "log" {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		if event.Message != string(line) {
			t.Fatalf("unexpected log message: %s", event.Message)
		}
	})
}

func TestExecuteInjectsSettingsAndMapsStatusEvents(t *testing.T) {
	python := requirePython(t)
	plugin := writePlugin(t, `
import json, sys
payload = json.loads(sys.stdin.read() or "{}")
result = {
  "settings": payload.get("context", {}).get("settings", {}),
  "source": payload.get("context", {}).get("source", "")
}
print(json.dumps({"status":"success","result":result}), flush=True)
`)

	service := New(stubRepo{plugin: plugin}, python, "").WithSettingsStore(stubSettingsStore{
		values: map[string]json.RawMessage{
			"plugin_settings:demo": json.RawMessage(`{"api_key":"k-demo"}`),
		},
	})

	var events []plugindomain.ExecutionEvent
	err := service.Execute(context.Background(), "demo", plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: "VERIFY",
		Context: map[string]any{
			"source": "account-execute",
		},
	}, func(event plugindomain.ExecutionEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("execute plugin: %v", err)
	}

	resultEvent, ok := findEvent(events, "result")
	if !ok {
		t.Fatalf("expected result event, got %#v", events)
	}

	settingsAny, ok := resultEvent.Data["settings"]
	if !ok {
		t.Fatalf("result settings not found: %#v", resultEvent.Data)
	}
	settingsMap, ok := settingsAny.(map[string]any)
	if !ok {
		t.Fatalf("result settings has unexpected type: %T", settingsAny)
	}
	if settingsMap["api_key"] != "k-demo" {
		t.Fatalf("unexpected settings value: %#v", settingsMap)
	}
}

func TestExecuteInjectsEmptySettingsWhenMissing(t *testing.T) {
	python := requirePython(t)
	plugin := writePlugin(t, `
import json, sys
payload = json.loads(sys.stdin.read() or "{}")
print(json.dumps({"status":"success","result":{"settings": payload.get("context", {}).get("settings", None)}}), flush=True)
`)

	service := New(stubRepo{plugin: plugin}, python, "").WithSettingsStore(stubSettingsStore{
		values: map[string]json.RawMessage{},
	})

	var events []plugindomain.ExecutionEvent
	err := service.Execute(context.Background(), "demo", plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: "VERIFY",
	}, func(event plugindomain.ExecutionEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("execute plugin: %v", err)
	}

	resultEvent, ok := findEvent(events, "result")
	if !ok {
		t.Fatalf("expected result event, got %#v", events)
	}

	settingsAny, ok := resultEvent.Data["settings"]
	if !ok {
		t.Fatalf("result settings not found: %#v", resultEvent.Data)
	}
	settingsMap, ok := settingsAny.(map[string]any)
	if !ok {
		t.Fatalf("result settings has unexpected type: %T", settingsAny)
	}
	if len(settingsMap) != 0 {
		t.Fatalf("expected empty settings object, got %#v", settingsMap)
	}
}

func TestExecuteTimeoutEmitsTimeoutError(t *testing.T) {
	python := requirePython(t)
	plugin := writePlugin(t, `
import time
time.sleep(1.0)
print('{"status":"success","result":{"ok":true}}', flush=True)
`)

	service := New(stubRepo{plugin: plugin}, python, "").WithExecutionTimeouts(ExecutionTimeouts{
		Account: 100 * time.Millisecond,
		Job:     150 * time.Millisecond,
		Agent:   0,
	})

	var events []plugindomain.ExecutionEvent
	err := service.Execute(context.Background(), "demo", plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: "WATCH",
		Context: map[string]any{
			"source": "manual",
		},
	}, func(event plugindomain.ExecutionEvent) {
		events = append(events, event)
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected timeout error: %v", err)
	}

	timeoutEvent, ok := findEvent(events, "error")
	if !ok {
		t.Fatalf("expected timeout error event, got %#v", events)
	}
	if timeoutEvent.Error != "TIMEOUT" {
		t.Fatalf("unexpected timeout error code: %s", timeoutEvent.Error)
	}
}

func TestExecuteAcceptsLargeJSONLines(t *testing.T) {
	python := requirePython(t)
	plugin := writePlugin(t, `
import json
payload = {"status":"success","result":{"blob":"x"*90000}}
print(json.dumps(payload), flush=True)
`)

	service := New(stubRepo{plugin: plugin}, python, "")

	var events []plugindomain.ExecutionEvent
	err := service.Execute(context.Background(), "demo", plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: "BIG_RESULT",
	}, func(event plugindomain.ExecutionEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("execute large output plugin: %v", err)
	}

	resultEvent, ok := findEvent(events, "result")
	if !ok {
		t.Fatalf("expected result event, got %#v", events)
	}
	blob, ok := resultEvent.Data["blob"].(string)
	if !ok {
		t.Fatalf("unexpected blob type: %T", resultEvent.Data["blob"])
	}
	if len(blob) != 90000 {
		t.Fatalf("unexpected blob length: %d", len(blob))
	}
}

type stubRepo struct {
	plugin plugindomain.Plugin
}

func (r stubRepo) List(ctx context.Context) ([]plugindomain.Plugin, error) {
	_ = ctx
	return []plugindomain.Plugin{r.plugin}, nil
}

func (r stubRepo) Get(ctx context.Context, key string) (*plugindomain.Plugin, error) {
	_ = ctx
	if key != r.plugin.Manifest.Key {
		return nil, fmt.Errorf("plugin not found: %s", key)
	}
	item := r.plugin
	return &item, nil
}

type stubSettingsStore struct {
	values map[string]json.RawMessage
}

func (s stubSettingsStore) GetConfig(ctx context.Context, key string) (json.RawMessage, error) {
	_ = ctx
	if s.values == nil {
		return json.RawMessage("{}"), nil
	}
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return json.RawMessage("{}"), nil
}

func requirePython(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found")
	}
	return path
}

func writePlugin(t *testing.T, script string) plugindomain.Plugin {
	t.Helper()

	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "main.py")
	if err := os.WriteFile(entrypoint, []byte(strings.TrimSpace(script)+"\n"), 0o644); err != nil {
		t.Fatalf("write plugin script: %v", err)
	}
	return plugindomain.Plugin{
		Manifest: plugindomain.Manifest{
			Key:        "demo",
			Entrypoint: "main.py",
		},
		Directory: dir,
		Healthy:   true,
	}
}

func findEvent(events []plugindomain.ExecutionEvent, eventType string) (plugindomain.ExecutionEvent, bool) {
	for _, item := range events {
		if item.Type == eventType {
			return item, true
		}
	}
	return plugindomain.ExecutionEvent{}, false
}
