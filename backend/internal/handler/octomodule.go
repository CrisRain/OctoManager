package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type OctoModuleHandler struct {
	svc service.OctoModuleService
}

func NewOctoModuleHandler(svc service.OctoModuleService) *OctoModuleHandler {
	return &OctoModuleHandler{svc: svc}
}

func (h *OctoModuleHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *OctoModuleHandler) Sync(c *gin.Context) {
	result, err := h.svc.SyncMissing(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OctoModuleHandler) Get(c *gin.Context) {
	typeKey := c.Param("typeKey")
	item, err := h.svc.Get(c.Request.Context(), typeKey)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *OctoModuleHandler) GetScript(c *gin.Context) {
	typeKey := c.Param("typeKey")
	content, err := h.svc.GetScript(c.Request.Context(), typeKey)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, content)
}

func (h *OctoModuleHandler) UpdateScript(c *gin.Context) {
	typeKey := c.Param("typeKey")
	var req dto.UpdateOctoModuleScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.UpdateScript(c.Request.Context(), typeKey, &req); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

func (h *OctoModuleHandler) ListFiles(c *gin.Context) {
	typeKey := c.Param("typeKey")
	result, err := h.svc.ListFiles(c.Request.Context(), typeKey)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OctoModuleHandler) GetFile(c *gin.Context) {
	typeKey := c.Param("typeKey")
	filename := strings.TrimPrefix(c.Param("filename"), "/")
	result, err := h.svc.GetFile(c.Request.Context(), typeKey, filename)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OctoModuleHandler) UpdateFile(c *gin.Context) {
	typeKey := c.Param("typeKey")
	filename := strings.TrimPrefix(c.Param("filename"), "/")
	var req dto.UpdateOctoModuleFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.UpdateFile(c.Request.Context(), typeKey, filename, &req); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

func (h *OctoModuleHandler) GetRunHistory(c *gin.Context) {
	typeKey := c.Param("typeKey")
	limit, offset, err := resolvePagination(c, 20, 200)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	result, err := h.svc.GetRunHistory(c.Request.Context(), typeKey, limit, offset)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OctoModuleHandler) GetVenvInfo(c *gin.Context) {
	typeKey := c.Param("typeKey")
	result, err := h.svc.GetVenvInfo(c.Request.Context(), typeKey)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OctoModuleHandler) InstallDeps(c *gin.Context) {
	typeKey := c.Param("typeKey")
	var req dto.InstallDepsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.InstallDeps(c.Request.Context(), typeKey, &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OctoModuleHandler) Action(c *gin.Context) {
	rawTypeKey := c.Param("typeKey")
	if typeKey, ok := parseColonActionParam(rawTypeKey, "ensure"); ok {
		result, err := h.svc.EnsureByTypeKey(c.Request.Context(), typeKey)
		if err != nil {
			response.FailWithError(c, err)
			return
		}
		response.Success(c, result)
		return
	}

	typeKey, ok := parseColonActionParam(rawTypeKey, "dry-run")
	if !ok {
		response.Fail(c, http.StatusBadRequest, "invalid path: expected /api/v1/octo-modules/{typeKey}:ensure or /api/v1/octo-modules/{typeKey}:dry-run")
		return
	}

	var req dto.OctoModuleDryRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.DryRun(c.Request.Context(), typeKey, &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}
