package grpcclient

import (
	"fmt"
	"sort"
	"strings"
)

// PluginRegistry resolves a plugin key to a gRPC server address (host:port).
type PluginRegistry interface {
	Address(pluginKey string) (string, error)
	Keys() []string
}

// StaticRegistry holds a fixed map of plugin key → address, configured at
// startup from environment variables or config file entries.
//
// Address resolution rules:
//   - Exact match on the plugin key (case-insensitive).
//   - Keys with hyphens or underscores are normalised to underscores before lookup.
type StaticRegistry struct {
	entries map[string]string // normalised key → "host:port"
}

// NewStaticRegistry builds a registry from a map of raw plugin keys to addresses.
func NewStaticRegistry(services map[string]PluginServiceConfig) *StaticRegistry {
	entries := make(map[string]string, len(services))
	for key, cfg := range services {
		if strings.TrimSpace(cfg.Address) == "" {
			continue
		}
		entries[normaliseKey(key)] = strings.TrimSpace(cfg.Address)
	}
	return &StaticRegistry{entries: entries}
}

// Address returns the gRPC server address for the given plugin key.
func (r *StaticRegistry) Address(pluginKey string) (string, error) {
	addr, ok := r.entries[normaliseKey(pluginKey)]
	if !ok || addr == "" {
		return "", fmt.Errorf("grpcclient: no address registered for plugin %q", pluginKey)
	}
	return addr, nil
}

// Keys returns all registered plugin keys (normalised).
func (r *StaticRegistry) Keys() []string {
	keys := make([]string, 0, len(r.entries))
	for k := range r.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// PluginServiceConfig holds the connection parameters for one plugin service.
type PluginServiceConfig struct {
	Address string // host:port — required
}

func normaliseKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
}
