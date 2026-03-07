package service

import "strings"

func normalizeScopeList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func normalizeOutlookTenant(tenant string) string {
	trimmed := trim(tenant)
	if trimmed == "" {
		return "common"
	}
	return trimmed
}
