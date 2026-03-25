package systemtransport

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	systemapp "octomanger/internal/domains/system/app"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	adminKey string
	service  systemapp.Service
}

func NewHandler(adminKey string, service systemapp.Service) Handler {
	return Handler{adminKey: adminKey, service: service}
}

// Register registers healthz on root and system routes on v2 group.
func (h Handler) Register(root *route.RouterGroup, v2 *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	root.GET("/healthz", h.healthz)
	v2.GET("/system/status", h.status)
	v2.GET("/dashboard", guard, h.dashboard)
	v2.GET("/config/:key", guard, h.getConfig)
	v2.PUT("/config/:key", guard, h.putConfig)
}

func (h Handler) healthz(ctx context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, map[string]any{"status": "ok"})
}

func (h Handler) dashboard(ctx context.Context, c *app.RequestContext) {
	summary, err := h.service.DashboardSummary(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h Handler) status(ctx context.Context, c *app.RequestContext) {
	status, err := h.service.Status(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, status)
}
