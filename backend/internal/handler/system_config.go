package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type SystemConfigHandler struct {
	svc service.SystemConfigService
}

func NewSystemConfigHandler(svc service.SystemConfigService) *SystemConfigHandler {
	return &SystemConfigHandler{svc: svc}
}

func (h *SystemConfigHandler) Get(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Fail(c, http.StatusBadRequest, "key is required")
		return
	}
	value, err := h.svc.Get(c.Request.Context(), key)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"key": key, "value": value})
}

func (h *SystemConfigHandler) Set(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Fail(c, http.StatusBadRequest, "key is required")
		return
	}
	var body struct {
		Value json.RawMessage `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.Set(c.Request.Context(), key, body.Value); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"key": key})
}
