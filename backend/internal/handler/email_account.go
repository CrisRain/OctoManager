package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type EmailAccountHandler struct {
	svc service.EmailAccountService
}

func NewEmailAccountHandler(svc service.EmailAccountService) *EmailAccountHandler {
	return &EmailAccountHandler{svc: svc}
}

func (h *EmailAccountHandler) List(c *gin.Context) {
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

func (h *EmailAccountHandler) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EmailAccountHandler) Create(c *gin.Context) {
	var req dto.CreateEmailAccountRequest
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

func (h *EmailAccountHandler) Patch(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	var req dto.PatchEmailAccountRequest
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

func (h *EmailAccountHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *EmailAccountHandler) Verify(c *gin.Context) {
	rawID := c.Param("id")
	idValue, ok := parseColonActionParam(rawID, "verify")
	if !ok {
		response.Fail(c, http.StatusBadRequest, "invalid path: expected /api/v1/email/accounts/{id}:verify")
		return
	}
	parsed, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil || parsed == 0 {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	item, err := h.svc.Verify(c.Request.Context(), parsed)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EmailAccountHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchEmailAccountIDsRequest
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

func (h *EmailAccountHandler) BatchVerify(c *gin.Context) {
	var req dto.BatchEmailAccountIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.BatchVerify(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) BatchImportGraph(c *gin.Context) {
	var req dto.BatchImportGraphEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.BatchImportGraph(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) BatchRegister(c *gin.Context) {
	var req dto.BatchRegisterEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.BatchRegister(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) ListMessages(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	var query dto.ListEmailMessagesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.ListMessages(c.Request.Context(), id, &query)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) GetMessage(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	messageID := strings.TrimSpace(c.Param("messageId"))
	if messageID == "" {
		response.Fail(c, http.StatusBadRequest, "invalid message id")
		return
	}
	var query dto.GetEmailMessageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.GetMessage(c.Request.Context(), id, query.Mailbox, messageID)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) ListMailboxes(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	var query dto.ListEmailMailboxesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.ListMailboxes(c.Request.Context(), id, &query)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) GetLatestMessage(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	var query dto.ListEmailMessagesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.GetLatestMessage(c.Request.Context(), id, &query)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) PreviewLatestMessage(c *gin.Context) {
	var req dto.PreviewEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.PreviewLatestMessage(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) PreviewMailboxes(c *gin.Context) {
	var req dto.PreviewEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.PreviewMailboxes(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) BuildOutlookAuthorizeURL(c *gin.Context) {
	var req dto.OutlookAuthorizeURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.BuildOutlookAuthorizeURL(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) ExchangeOutlookCode(c *gin.Context) {
	var req dto.OutlookExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.ExchangeOutlookCode(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *EmailAccountHandler) RefreshOutlookToken(c *gin.Context) {
	var req dto.OutlookRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.RefreshOutlookToken(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}
