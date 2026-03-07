package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type webhookKeyChecker interface {
	HasAnyKey(ctx context.Context) (bool, error)
	ValidateWebhookKey(ctx context.Context, rawKey string, slug string) (*dto.ApiKeyResponse, error)
}

type TriggerHandler struct {
	svc       service.TriggerService
	apiKeySvc webhookKeyChecker
}

func NewTriggerHandler(svc service.TriggerService) *TriggerHandler {
	return &TriggerHandler{svc: svc}
}

func NewTriggerHandlerWithAuth(svc service.TriggerService, apiKeySvc webhookKeyChecker) *TriggerHandler {
	return &TriggerHandler{svc: svc, apiKeySvc: apiKeySvc}
}

func (h *TriggerHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *TriggerHandler) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid trigger id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *TriggerHandler) Create(c *gin.Context) {
	var req dto.CreateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *TriggerHandler) Patch(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid trigger id")
		return
	}
	var req dto.PatchTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Patch(c.Request.Context(), id, &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *TriggerHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid trigger id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *TriggerHandler) Fire(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		response.Fail(c, http.StatusBadRequest, "slug is required")
		return
	}

	var payload dto.FireTriggerRequest
	var req *dto.FireTriggerRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		if !errors.Is(err, io.EOF) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		req = &payload
	}

	// Try X-Api-Key webhook auth first.
	if apiKey := strings.TrimSpace(c.GetHeader("X-Api-Key")); apiKey != "" && h.apiKeySvc != nil {
		if _, err := h.apiKeySvc.ValidateWebhookKey(c.Request.Context(), apiKey, slug); err != nil {
			response.Fail(c, http.StatusUnauthorized, "invalid or insufficient API key")
			return
		}
		result, err := h.svc.FireDirect(c.Request.Context(), slug, req)
		if err != nil {
			response.FailWithError(c, err)
			return
		}
		response.Success(c, result)
		return
	}

	// Fall back to per-trigger bearer token.
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	rawToken := ""
	if len(authHeader) >= 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		rawToken = strings.TrimSpace(authHeader[7:])
	}
	if rawToken == "" {
		response.Fail(c, http.StatusUnauthorized, "missing or invalid Authorization header")
		return
	}

	result, err := h.svc.Fire(c.Request.Context(), slug, rawToken, req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}
