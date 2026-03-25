package plugintransport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypedomain "octomanger/internal/domains/account-types/domain"
	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

// ConfigStore is the subset of systemapp.Service used for plugin settings persistence.
type ConfigStore interface {
	GetConfig(ctx context.Context, key string) (json.RawMessage, error)
	SetConfig(ctx context.Context, key string, value json.RawMessage) error
}

type Handler struct {
	adminKey     string
	service      plugins.PluginService
	accountTypes accounttypeapp.Service
	configs      ConfigStore
}

func NewHandler(adminKey string, service plugins.PluginService, accountTypes accounttypeapp.Service, configs ConfigStore) Handler {
	return Handler{adminKey: adminKey, service: service, accountTypes: accountTypes, configs: configs}
}

func (h Handler) Register(r *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	r.GET("/plugins", h.list)
	r.GET("/plugins/:key", h.get)
	r.POST("/plugins/sync", guard, h.sync)
	r.GET("/plugins/:key/settings", guard, h.getSettings)
	r.PUT("/plugins/:key/settings", guard, h.putSettings)
	r.POST("/plugins/:key/actions/:action", guard, h.executeAction)
}

func (h Handler) list(ctx context.Context, c *app.RequestContext) {
	plugins, err := h.service.List(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": plugins})
}

func (h Handler) get(ctx context.Context, c *app.RequestContext) {
	plugin, err := h.service.Get(ctx, c.Param("key"))
	if err != nil {
		httpx.NotFound(ctx, c, err.Error())
		return
	}
	if plugin == nil {
		httpx.NotFound(ctx, c, "plugin not found")
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h Handler) sync(ctx context.Context, c *app.RequestContext) {
	var synced, failed int
	var errs []string

	err := h.service.SyncAccountTypes(ctx, func(ctx context.Context, spec pluginapp.AccountTypeSpec) error {
		_, err := h.accountTypes.Upsert(ctx, accounttypedomain.CreateInput{
			Key:          spec.Key,
			Name:         spec.Name,
			Category:     spec.Category,
			Schema:       spec.Schema,
			Capabilities: spec.Capabilities,
		})
		if err != nil {
			failed++
			errs = append(errs, spec.Key+": "+err.Error())
			return nil
		}
		synced++
		return nil
	})
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"synced": synced,
		"failed": failed,
		"errors": errs,
	})
}

func (h Handler) getSettings(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")

	if _, err := h.service.Get(ctx, key); err != nil {
		httpx.NotFound(ctx, c, err.Error())
		return
	}

	raw, err := h.configs.GetConfig(ctx, settingsKey(key))
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}

	if len(raw) == 0 || string(raw) == "null" {
		c.JSON(http.StatusOK, map[string]any{})
		return
	}

	settings := map[string]any{}
	if err := json.Unmarshal(raw, &settings); err != nil {
		httpx.InternalServerError(ctx, c, "invalid plugin settings JSON")
		return
	}
	if settings == nil {
		settings = map[string]any{}
	}
	c.JSON(http.StatusOK, settings)
}

func (h Handler) putSettings(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")

	if _, err := h.service.Get(ctx, key); err != nil {
		httpx.NotFound(ctx, c, err.Error())
		return
	}

	body := c.Request.Body()
	if len(body) == 0 {
		body = []byte("{}")
	}
	if !json.Valid(body) {
		httpx.BadRequest(ctx, c, "body must be valid JSON")
		return
	}

	var settings map[string]any
	if err := json.Unmarshal(body, &settings); err != nil {
		httpx.BadRequest(ctx, c, "body must be a JSON object")
		return
	}
	if settings == nil {
		settings = map[string]any{}
	}
	normalizedBody, err := json.Marshal(settings)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}

	if err := h.configs.SetConfig(ctx, settingsKey(key), json.RawMessage(normalizedBody)); err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"saved": true})
}

func settingsKey(pluginKey string) string {
	return "plugin_settings:" + pluginKey
}

func (h Handler) executeAction(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	action := c.Param("action")

	plugin, err := h.service.Get(ctx, key)
	if err != nil {
		httpx.NotFound(ctx, c, err.Error())
		return
	}
	if plugin == nil {
		httpx.NotFound(ctx, c, "plugin not found")
		return
	}

	var payload struct {
		Params map[string]any `json:"params"`
		Spec   map[string]any `json:"spec"`
	}
	if err := c.BindJSON(&payload); err != nil {
		httpx.BadRequest(ctx, c, "invalid request body")
		return
	}

	req := plugindomain.ExecutionRequest{
		Action: action,
		Input: map[string]any{
			"params": payload.Params,
			"spec":   payload.Spec,
		},
		Mode: "sync",
	}

	var result plugindomain.ExecutionEvent
	var resultErr error

	err = h.service.Execute(ctx, key, req, func(event plugindomain.ExecutionEvent) {
		if event.Type == "result" {
			result = event
		} else if event.Type == "error" {
			resultErr = fmt.Errorf("%s (code: %s)", event.Message, event.Error)
		}
	})

	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}

	if resultErr != nil {
		httpx.InternalServerError(ctx, c, resultErr.Error())
		return
	}

	if result.Type == "result" {
		c.JSON(http.StatusOK, map[string]any{
			"message": result.Message,
			"data":    result.Data,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"message": "执行成功，但未返回结果",
	})
}
