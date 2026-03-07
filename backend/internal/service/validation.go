package service

import (
	"encoding/json"
	"strings"
)

func trim(value string) string {
	return strings.TrimSpace(value)
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	_, ok := value.(map[string]any)
	return ok
}

func isJSONObjectOrNull(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	if value == nil {
		return true
	}
	_, ok := value.(map[string]any)
	return ok
}

func isValidCategory(category string) bool {
	switch category {
	case "system", "email", "generic":
		return true
	default:
		return false
	}
}

func isGenericCategory(category string) bool {
	return strings.EqualFold(trim(category), "generic")
}

func isValidJobStatus(status int16) bool {
	switch status {
	case 0, 1, 2, 3, 4:
		return true
	default:
		return false
	}
}
