package triggertransport

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	triggerapp "octomanger/internal/domains/triggers/app"
	triggerdomain "octomanger/internal/domains/triggers/domain"
	triggerpostgres "octomanger/internal/domains/triggers/infra/postgres"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	adminKey string
	service  triggerapp.Service
}

func NewHandler(adminKey string, service triggerapp.Service) Handler {
	return Handler{adminKey: adminKey, service: service}
}

// Register registers trigger routes on the v2 group.
// Webhooks are registered on root to avoid the /api/v2 prefix.
func (h Handler) Register(v2 *route.RouterGroup, root *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	v2.GET("/triggers", h.list)
	v2.POST("/triggers", guard, h.create)
	v2.GET("/triggers/:id", h.get)
	v2.PATCH("/triggers/:id", guard, h.patch)
	v2.DELETE("/triggers/:id", guard, h.delete)
	v2.POST("/triggers/:id/fire", guard, h.fireByID)
	root.POST("/api/v2/webhooks/:key", h.fireByKey) // no auth guard — token in header
}

func (h Handler) get(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid trigger id")
		return
	}
	item, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, triggerpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "trigger not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) list(ctx context.Context, c *app.RequestContext) {
	items, err := h.service.List(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h Handler) create(ctx context.Context, c *app.RequestContext) {
	var input triggerdomain.CreateInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	result, err := h.service.Create(ctx, input)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h Handler) patch(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid trigger id")
		return
	}
	var input triggerdomain.PatchTriggerInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Patch(ctx, id, input)
	if err != nil {
		if errors.Is(err, triggerpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "trigger not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) delete(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid trigger id")
		return
	}
	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, triggerpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "trigger not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"deleted": true})
}

func (h Handler) fireByID(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid trigger id")
		return
	}
	input, err := decodeOptionalBody(c)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	result, err := h.service.FireByID(ctx, id, input)
	if err != nil {
		if errors.Is(err, triggerpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "trigger not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) fireByKey(ctx context.Context, c *app.RequestContext) {
	input, err := decodeOptionalBody(c)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	token := strings.TrimSpace(string(c.GetHeader("X-Trigger-Token")))
	if token == "" {
		token = strings.TrimSpace(strings.TrimPrefix(string(c.GetHeader("Authorization")), "Bearer "))
	}
	result, err := h.service.FireByKey(ctx, c.Param("key"), token, input)
	if err != nil {
		if errors.Is(err, triggerpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "trigger not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func decodeOptionalBody(c *app.RequestContext) (map[string]any, error) {
	if c.Request.Header.ContentLength() == 0 {
		return map[string]any{}, nil
	}
	result := map[string]any{}
	if err := httpx.DecodeJSON(c, &result); err != nil {
		return nil, err
	}
	return result, nil
}
