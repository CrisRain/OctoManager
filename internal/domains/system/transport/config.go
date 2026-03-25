package systemtransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"octomanger/internal/platform/httpx"
)

func (h Handler) getConfig(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	if key == "" {
		httpx.BadRequest(ctx, c, "missing configuration key")
		return
	}
	val, err := h.service.GetConfig(ctx, key)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"key": key, "value": val})
}

func (h Handler) putConfig(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	if key == "" {
		httpx.BadRequest(ctx, c, "missing configuration key")
		return
	}
	var req struct {
		Value json.RawMessage `json:"value"`
	}
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.BadRequest(ctx, c, "invalid JSON body")
		return
	}
	if err := h.service.SetConfig(ctx, key, req.Value); err != nil {
		if strings.Contains(err.Error(), "must be valid JSON") {
			httpx.BadRequest(ctx, c, err.Error())
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	val, _ := h.service.GetConfig(ctx, key)
	c.JSON(http.StatusOK, map[string]any{"key": key, "value": val})
}
