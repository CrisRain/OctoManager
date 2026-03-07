package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/pkg/response"
)

// AdminKeyChecker is satisfied by services.ApiKeyService.
type AdminKeyChecker interface {
	HasAnyAdminKey(ctx context.Context) (bool, error)
	ValidateAdminKey(ctx context.Context, rawKey string) (*dto.ApiKeyResponse, error)
}

// ApiKeyChecker is satisfied by services.ApiKeyService (kept for trigger webhook use).
type ApiKeyChecker interface {
	HasAnyKey(ctx context.Context) (bool, error)
	ValidateKey(ctx context.Context, rawKey string) (*dto.ApiKeyResponse, error)
}

// AdminKeyAuth enforces admin API key authentication on /api/v1/* routes.
// If no admin keys exist in the database, all requests are allowed (bootstrap mode).
func AdminKeyAuth(checker AdminKeyChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			c.Next()
			return
		}
		hasKey, err := checker.HasAnyAdminKey(c.Request.Context())
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, "failed to validate api keys")
			return
		}
		if !hasKey {
			c.Next()
			return
		}
		rawKey := c.GetHeader("X-Api-Key")
		if rawKey == "" {
			response.Fail(c, http.StatusUnauthorized, "X-Api-Key header is required")
			return
		}
		if _, err := checker.ValidateAdminKey(c.Request.Context(), rawKey); err != nil {
			response.Fail(c, http.StatusUnauthorized, "invalid or disabled API key")
			return
		}
		c.Next()
	}
}

// ApiKeyAuth is the legacy alias kept so existing call sites compile unchanged.
// It now delegates to AdminKeyAuth semantics.
func ApiKeyAuth(checker AdminKeyChecker) gin.HandlerFunc {
	return AdminKeyAuth(checker)
}
