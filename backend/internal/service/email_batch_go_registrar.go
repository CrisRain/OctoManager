package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"octomanger/backend/internal/dto"
)

// GoEmailBatchRegistrar generates email account candidates natively in Go,
// replacing the Python bridge for address generation.
type GoEmailBatchRegistrar struct{}

func NewGoEmailBatchRegistrar() *GoEmailBatchRegistrar {
	return &GoEmailBatchRegistrar{}
}

func (r *GoEmailBatchRegistrar) Prepare(
	_ context.Context,
	input dto.BatchRegisterEmailRequest,
) (EmailBatchPreparedResult, error) {
	prefix := strings.TrimSpace(input.Prefix)
	if prefix == "" {
		prefix = "mail"
	}
	startIndex := input.StartIndex
	if startIndex <= 0 {
		startIndex = 1
	}
	mailbox := "INBOX"
	var graphOverride map[string]any
	if mb, ok := input.Options["mailbox"].(string); ok && strings.TrimSpace(mb) != "" {
		mailbox = strings.TrimSpace(mb)
	}
	if gc, ok := input.Options["graph_config"].(map[string]any); ok {
		graphOverride = gc
	}

	domain := strings.TrimSpace(input.Domain)
	provider := strings.TrimSpace(input.Provider)

	candidates := make([]BatchRegisterCandidate, 0, input.Count)
	for i := 0; i < input.Count; i++ {
		address := fmt.Sprintf("%s%d@%s", prefix, startIndex+i, domain)
		cfg := buildDefaultGraphConfig(address, mailbox)
		for k, v := range graphOverride {
			cfg[k] = v
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return EmailBatchPreparedResult{}, fmt.Errorf("marshal graph_config for %s: %w", address, err)
		}
		candidates = append(candidates, BatchRegisterCandidate{
			Address:     address,
			Provider:    provider,
			Status:      input.Status,
			GraphConfig: raw,
		})
	}

	return EmailBatchPreparedResult{
		Requested:  input.Count,
		Provider:   provider,
		Candidates: candidates,
	}, nil
}

func buildDefaultGraphConfig(address, mailbox string) map[string]any {
	tenant := resolveGraphTenant(address)
	return map[string]any{
		"auth_method":    "graph_oauth2",
		"username":       address,
		"tenant":         tenant,
		"scope":          []string{"https://graph.microsoft.com/.default"},
		"token_url":      fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
		"graph_base_url": "https://graph.microsoft.com/v1.0",
		"mailbox":        mailbox,
	}
}

func resolveGraphTenant(address string) string {
	parts := strings.SplitN(strings.ToLower(address), "@", 2)
	if len(parts) != 2 {
		return "common"
	}
	switch parts[1] {
	case "outlook.com", "hotmail.com", "live.com", "msn.com":
		return "consumers"
	default:
		return "common"
	}
}

var _ EmailBatchRegistrar = (*GoEmailBatchRegistrar)(nil)
