package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type SystemHandler struct {
	svc service.SystemService
}

func NewSystemHandler(svc service.SystemService) *SystemHandler {
	return &SystemHandler{svc: svc}
}

func (h *SystemHandler) Status(c *gin.Context) {
	result, err := h.svc.Status(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SystemHandler) Migrate(c *gin.Context) {
	result, err := h.svc.Migrate(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SystemHandler) Setup(c *gin.Context) {
	var req dto.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Setup(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}
