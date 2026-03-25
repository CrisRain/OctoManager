package agenttransport

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	agentapp "octomanger/internal/domains/agents/app"
	agentdomain "octomanger/internal/domains/agents/domain"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	adminKey string
	service  *agentapp.Service
}

func NewHandler(adminKey string, service *agentapp.Service) Handler {
	return Handler{adminKey: adminKey, service: service}
}

func (h Handler) Register(r *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	r.GET("/agents", h.list)
	r.POST("/agents", guard, h.create)
	r.GET("/agents/:id", h.get)
	r.PATCH("/agents/:id", guard, h.patch)
	r.DELETE("/agents/:id", guard, h.delete)
	r.GET("/agents/:id/status", h.status)
	r.POST("/agents/:id/start", guard, h.start)
	r.POST("/agents/:id/stop", guard, h.stop)
	r.GET("/agents/:id/events", h.streamEvents)
}

func (h Handler) get(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}
	item, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, agentpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "agent not found")
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
	var input agentdomain.CreateAgentInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Create(ctx, input)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h Handler) patch(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}
	var input agentdomain.PatchAgentInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Patch(ctx, id, input)
	if err != nil {
		if errors.Is(err, agentpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "agent not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) status(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}
	s, err := h.service.GetStatus(ctx, id)
	if err != nil {
		if errors.Is(err, agentpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "agent not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h Handler) start(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}
	if err := h.service.Start(ctx, id); err != nil {
		if errors.Is(err, agentpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "agent not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, map[string]any{"started": true})
}

func (h Handler) stop(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}
	if err := h.service.Stop(ctx, id); err != nil {
		if errors.Is(err, agentpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "agent not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, map[string]any{"stopped": true})
}

func (h Handler) delete(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}
	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, agentpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "agent not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) streamEvents(ctx context.Context, c *app.RequestContext) {
	agentID, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid agent id")
		return
	}

	httpx.PrepareStream(c, func(w *httpx.StreamWriter) {
		const (
			minPoll          = 200 * time.Millisecond
			maxPoll          = 2 * time.Second
			heartbeatInterval = 5 * time.Second
		)
		afterID := int64(0)
		wait := minPoll
		nextHeartbeatAt := time.Now().UTC()

		writeHeartbeat := func() error {
			now := time.Now().UTC()
			payload := map[string]any{
				"ts": now,
			}
			if status, statusErr := h.service.GetStatus(ctx, agentID); statusErr == nil && status != nil {
				payload["runtime_state"] = status.RuntimeState
				payload["desired_state"] = status.DesiredState
				payload["updated_at"] = status.UpdatedAt
				if status.LastError != "" {
					payload["last_error"] = status.LastError
				}
				if status.LastHeartbeatAt != nil {
					payload["last_heartbeat_at"] = status.LastHeartbeatAt
				}
			}
			return w.WriteEvent("heartbeat", payload)
		}

		if err := writeHeartbeat(); err != nil {
			return
		}
		nextHeartbeatAt = time.Now().UTC().Add(heartbeatInterval)

		for {
			logs, err := h.service.ListLogsAfter(ctx, agentID, afterID)
			if err != nil {
				_ = w.WriteEvent("error", map[string]any{"message": err.Error()})
				return
			}
			for _, item := range logs {
				afterID = item.ID
				if err := w.WriteEvent(item.EventType, item); err != nil {
					return
				}
			}
			if len(logs) > 0 {
				wait = minPoll
			} else {
				wait = min(time.Duration(float64(wait)*1.5), maxPoll)
			}

			now := time.Now().UTC()
			if !now.Before(nextHeartbeatAt) {
				if err := writeHeartbeat(); err != nil {
					return
				}
				nextHeartbeatAt = now.Add(heartbeatInterval)
			}

			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	})
}
