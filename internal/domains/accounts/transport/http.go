package accountstransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	accountapp "octomanger/internal/domains/accounts/app"
	accountdomain "octomanger/internal/domains/accounts/domain"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	plugindomain "octomanger/internal/domains/plugins/domain"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

// pluginExecutor is the narrow interface accounts transport needs.
type pluginExecutor interface {
	Execute(ctx context.Context, pluginKey string, request plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error
}

type Handler struct {
	adminKey string
	service  accountapp.Service
	plugins  pluginExecutor
}

func NewHandler(adminKey string, service accountapp.Service, plugins pluginExecutor) Handler {
	return Handler{adminKey: adminKey, service: service, plugins: plugins}
}

func (h Handler) Register(r *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	r.GET("/accounts", h.list)
	r.POST("/accounts", guard, h.create)
	r.GET("/accounts/:id", h.get)
	r.PATCH("/accounts/:id", guard, h.patch)
	r.DELETE("/accounts/:id", guard, h.delete)
	r.POST("/accounts/:id/execute", guard, h.execute)
}

func (h Handler) list(ctx context.Context, c *app.RequestContext) {
	items, err := h.service.List(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h Handler) get(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	item, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) create(ctx context.Context, c *app.RequestContext) {
	var input accountdomain.CreateInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Create(ctx, input)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h Handler) patch(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	var input accountdomain.PatchInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Patch(ctx, id, input)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) delete(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"deleted": true})
}

type executeBody struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

func (h Handler) execute(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	var body executeBody
	if err := httpx.DecodeJSON(c, &body); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	if body.Action == "" {
		httpx.BadRequest(ctx, c, "action is required")
		return
	}

	account, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	if account.AccountTypeKey == "" {
		httpx.BadRequest(ctx, c, "account has no account_type_key, cannot determine plugin")
		return
	}

	params := body.Params
	if params == nil {
		params = map[string]any{}
	}

	var (
		resultData   map[string]any
		errorCode    string
		errorMessage string
	)

	execErr := h.plugins.Execute(ctx, account.AccountTypeKey, plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: body.Action,
		Input: map[string]any{
			"account": map[string]any{
				"id":         account.ID,
				"identifier": account.Identifier,
				"spec":       account.Spec,
			},
			"params": params,
		},
		Context: map[string]any{
			"source": "account-execute",
		},
	}, func(event plugindomain.ExecutionEvent) {
		switch event.Type {
		case "result":
			resultData = event.Data
		case "error":
			errorCode = event.Error
			if errorCode == "" {
				errorCode = "PLUGIN_ERROR"
			}
			errorMessage = event.Message
		}
	})

	if execErr != nil && errorMessage == "" {
		errorCode = "EXECUTION_FAILED"
		errorMessage = execErr.Error()
	}

	if errorMessage != "" {
		c.JSON(http.StatusOK, map[string]any{
			"status":        "error",
			"error_code":    errorCode,
			"error_message": errorMessage,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"status": "ok",
		"result": resultData,
	})
}
