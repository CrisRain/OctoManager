package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"octomanger/internal/platform/config"
)

const pluginGRPCServicesConfigKey = "plugins.grpc_services"

type pluginConfigStore interface {
	GetConfig(ctx context.Context, key string) (json.RawMessage, error)
	SetConfig(ctx context.Context, key string, value json.RawMessage) error
}

func resolvePluginServices(
	ctx context.Context,
	store pluginConfigStore,
	defaults map[string]config.PluginServiceEntry,
) (map[string]config.PluginServiceEntry, error) {
	normalizedDefaults := normalizePluginServices(defaults)

	raw, err := store.GetConfig(ctx, pluginGRPCServicesConfigKey)
	if err != nil {
		return nil, fmt.Errorf("get plugin service config: %w", err)
	}

	current, err := decodePluginServices(raw)
	if err != nil {
		return nil, fmt.Errorf("decode plugin service config: %w", err)
	}

	changed := false
	if len(current) == 0 && len(normalizedDefaults) > 0 {
		current = make(map[string]config.PluginServiceEntry, len(normalizedDefaults))
		changed = true
	}

	for key, entry := range normalizedDefaults {
		if existing, ok := current[key]; ok && strings.TrimSpace(existing.Address) != "" {
			continue
		}
		current[key] = entry
		changed = true
	}

	if changed {
		payload, err := json.Marshal(current)
		if err != nil {
			return nil, fmt.Errorf("marshal plugin service config: %w", err)
		}
		if err := store.SetConfig(ctx, pluginGRPCServicesConfigKey, json.RawMessage(payload)); err != nil {
			return nil, fmt.Errorf("persist plugin service config: %w", err)
		}
	}

	return current, nil
}

func decodePluginServices(raw json.RawMessage) (map[string]config.PluginServiceEntry, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return map[string]config.PluginServiceEntry{}, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	services := make(map[string]config.PluginServiceEntry, len(payload))
	for key, value := range payload {
		normalizedKey := normalizePluginKey(key)
		if normalizedKey == "" {
			continue
		}

		switch item := value.(type) {
		case string:
			if address := strings.TrimSpace(item); address != "" {
				services[normalizedKey] = config.PluginServiceEntry{Address: address}
			}
		case map[string]any:
			if address := strings.TrimSpace(asString(item["address"])); address != "" {
				services[normalizedKey] = config.PluginServiceEntry{Address: address}
			}
		}
	}

	return services, nil
}

func normalizePluginServices(in map[string]config.PluginServiceEntry) map[string]config.PluginServiceEntry {
	out := make(map[string]config.PluginServiceEntry, len(in))
	for key, entry := range in {
		normalizedKey := normalizePluginKey(key)
		address := strings.TrimSpace(entry.Address)
		if normalizedKey == "" || address == "" {
			continue
		}
		out[normalizedKey] = config.PluginServiceEntry{Address: address}
	}
	return out
}

func normalizePluginKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
}

func asString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
