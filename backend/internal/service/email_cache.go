package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

func buildEmailCacheKey(prefix string, parts ...string) string {
	hasher := sha1.New()
	for _, part := range parts {
		_, _ = hasher.Write([]byte(part))
		_, _ = hasher.Write([]byte{0})
	}
	return "emailcache:" + strings.TrimSpace(prefix) + ":" + hex.EncodeToString(hasher.Sum(nil))
}

func (s *emailAccountService) getCachedJSON(ctx context.Context, key string, out any) bool {
	if s == nil || s.cacheClient == nil || strings.TrimSpace(key) == "" || out == nil {
		return false
	}
	value, err := s.cacheClient.Get(ctx, key).Bytes()
	if err != nil || len(value) == 0 {
		return false
	}
	if err := json.Unmarshal(value, out); err != nil {
		return false
	}
	return true
}

func (s *emailAccountService) setCachedJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if s == nil || s.cacheClient == nil || strings.TrimSpace(key) == "" || value == nil || ttl <= 0 {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = s.cacheClient.Set(ctx, key, raw, ttl).Err()
}
