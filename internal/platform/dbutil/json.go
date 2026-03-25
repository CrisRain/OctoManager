package dbutil

import (
	"encoding/json"
	"log/slog"
)

// DecodeJSONMap decodes a JSONB column value into a string-keyed map.
// Empty input returns an empty map. Malformed JSON is logged and an empty map is returned.
func DecodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	v := map[string]any{}
	if err := json.Unmarshal(raw, &v); err != nil {
		slog.Error("dbutil: corrupt JSON map in database column", "error", err)
	}
	return v
}

// NormalizeMap converts a nil map to an empty non-nil map.
func NormalizeMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

// MergeMaps merges override into a copy of base.
// Keys in override take precedence. Nil maps are treated as empty.
func MergeMaps(base, override map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(override))
	for k, v := range NormalizeMap(base) {
		merged[k] = v
	}
	for k, v := range NormalizeMap(override) {
		merged[k] = v
	}
	return merged
}
