package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type AccountTypeHandler struct {
	svc service.AccountTypeService
}

func NewAccountTypeHandler(svc service.AccountTypeService) *AccountTypeHandler {
	return &AccountTypeHandler{svc: svc}
}

func (h *AccountTypeHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *AccountTypeHandler) Get(c *gin.Context) {
	key := c.Param("key")
	item, err := h.svc.Get(c.Request.Context(), key)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *AccountTypeHandler) Create(c *gin.Context) {
	var req dto.CreateAccountTypeRequest
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

func (h *AccountTypeHandler) Patch(c *gin.Context) {
	key := c.Param("key")
	var req dto.PatchAccountTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Patch(c.Request.Context(), key, &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *AccountTypeHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
