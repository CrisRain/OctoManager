package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type JobHandler struct {
	svc service.JobService
}

func NewJobHandler(svc service.JobService) *JobHandler {
	return &JobHandler{svc: svc}
}

func (h *JobHandler) List(c *gin.Context) {
	limit, offset, err := resolvePagination(c, 50, 500)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	result, err := h.svc.ListPaged(c.Request.Context(), limit, offset)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *JobHandler) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *JobHandler) Create(c *gin.Context) {
	var req dto.CreateJobRequest
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

func (h *JobHandler) Patch(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var req dto.PatchJobRequest
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

func (h *JobHandler) Cancel(c *gin.Context) {
	rawID := c.Param("id")
	idValue, ok := parseColonActionParam(rawID, "cancel")
	if !ok {
		response.Fail(c, http.StatusBadRequest, "invalid path: expected /api/v1/jobs/{id}:cancel")
		return
	}
	parsed, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil || parsed == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	item, err := h.svc.Cancel(c.Request.Context(), parsed)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *JobHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
