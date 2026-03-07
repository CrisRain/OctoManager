package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type AccountHandler struct {
	svc service.AccountService
}

func NewAccountHandler(svc service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) List(c *gin.Context) {
	limit, offset, err := resolvePagination(c, 20, 200)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	typeKey := c.Query("type_key")
	result, err := h.svc.ListPaged(c.Request.Context(), limit, offset, typeKey)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AccountHandler) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *AccountHandler) Create(c *gin.Context) {
	var req dto.CreateAccountRequest
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

func (h *AccountHandler) Patch(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	var req dto.PatchAccountRequest
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

func (h *AccountHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *AccountHandler) BatchPatch(c *gin.Context) {
	var req dto.BatchPatchAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.BatchPatch(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AccountHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.BatchDelete(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}
