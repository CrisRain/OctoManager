package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"octomanger/internal/platform/config"
)

func TestResolvePluginServicesInitializesDefaultsIntoStore(t *testing.T) {
	store := &stubPluginConfigStore{
		values: map[string]json.RawMessage{},
	}

	services, err := resolvePluginServices(context.Background(), store, map[string]config.PluginServiceEntry{
		"octo_demo": {Address: "127.0.0.1:50051"},
	})
	if err != nil {
		t.Fatalf("resolve plugin services: %v", err)
	}

	if services["octo_demo"].Address != "127.0.0.1:50051" {
		t.Fatalf("unexpected resolved address %q", services["octo_demo"].Address)
	}

	saved := store.values[pluginGRPCServicesConfigKey]
	if len(saved) == 0 {
		t.Fatalf("expected plugin services config to be initialized in store")
	}
}

func TestResolvePluginServicesUsesDatabaseValueAndBackfillsMissingDefaults(t *testing.T) {
	store := &stubPluginConfigStore{
		values: map[string]json.RawMessage{
			pluginGRPCServicesConfigKey: json.RawMessage(`{
				"octo_demo": {"address": "127.0.0.1:60051"}
			}`),
		},
	}

	services, err := resolvePluginServices(context.Background(), store, map[string]config.PluginServiceEntry{
		"octo_demo":  {Address: "127.0.0.1:50051"},
		"other_demo": {Address: "127.0.0.1:50052"},
	})
	if err != nil {
		t.Fatalf("resolve plugin services: %v", err)
	}

	if services["octo_demo"].Address != "127.0.0.1:60051" {
		t.Fatalf("expected database value to win, got %q", services["octo_demo"].Address)
	}
	if services["other_demo"].Address != "127.0.0.1:50052" {
		t.Fatalf("expected missing default to be backfilled, got %q", services["other_demo"].Address)
	}
}

type stubPluginConfigStore struct {
	values map[string]json.RawMessage
}

func (s *stubPluginConfigStore) GetConfig(ctx context.Context, key string) (json.RawMessage, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return json.RawMessage(`{}`), nil
}

func (s *stubPluginConfigStore) SetConfig(ctx context.Context, key string, value json.RawMessage) error {
	if s.values == nil {
		s.values = map[string]json.RawMessage{}
	}
	s.values[key] = append(json.RawMessage(nil), value...)
	return nil
}
