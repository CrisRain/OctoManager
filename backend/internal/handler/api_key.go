package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type ApiKeyHandler struct {
	svc service.ApiKeyService
}

func NewApiKeyHandler(svc service.ApiKeyService) *ApiKeyHandler {
	return &ApiKeyHandler{svc: svc}
}

func (h *ApiKeyHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ApiKeyHandler) Create(c *gin.Context) {
	var req dto.CreateApiKeyRequest
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

func (h *ApiKeyHandler) SetEnabled(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid api key id")
		return
	}
	var req dto.SetApiKeyEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.SetEnabled(c.Request.Context(), id, req.Enabled)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ApiKeyHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid api key id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
