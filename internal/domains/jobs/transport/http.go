package jobtransport

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	jobapp "octomanger/internal/domains/jobs/app"
	jobdomain "octomanger/internal/domains/jobs/domain"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	adminKey string
	service  jobapp.Service
}

func NewHandler(adminKey string, service jobapp.Service) Handler {
	return Handler{adminKey: adminKey, service: service}
}

func (h Handler) Register(r *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	r.GET("/job-definitions", h.listDefinitions)
	r.POST("/job-definitions", guard, h.createDefinition)
	r.GET("/job-definitions/:id", h.getDefinition)
	r.PATCH("/job-definitions/:id", guard, h.patchDefinition)
	r.DELETE("/job-definitions/:id", guard, h.deleteDefinition)
	r.POST("/job-definitions/:id/executions", guard, h.enqueueExecution)
	r.GET("/job-executions", h.listExecutions)
	r.GET("/job-executions/:id", h.getExecution)
	r.GET("/job-executions/:id/events", h.streamExecutionEvents)
}

func (h Handler) getDefinition(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid job definition id")
		return
	}
	item, err := h.service.GetDefinition(ctx, id)
	if err != nil {
		if errors.Is(err, jobpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "job definition not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) listDefinitions(ctx context.Context, c *app.RequestContext) {
	items, err := h.service.ListDefinitions(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h Handler) createDefinition(ctx context.Context, c *app.RequestContext) {
	var input jobdomain.CreateDefinitionInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.CreateDefinition(ctx, input)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h Handler) patchDefinition(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid job definition id")
		return
	}
	var input jobdomain.PatchDefinitionInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.PatchDefinition(ctx, id, input)
	if err != nil {
		if errors.Is(err, jobpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "job definition not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) deleteDefinition(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid job definition id")
		return
	}
	if err := h.service.DeleteDefinition(ctx, id); err != nil {
		if errors.Is(err, jobpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "job definition not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) enqueueExecution(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid job definition id")
		return
	}
	item, err := h.service.EnqueueExecution(ctx, id, "api:user", "manual", nil)
	if err != nil {
		if errors.Is(err, jobpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "job definition not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h Handler) listExecutions(ctx context.Context, c *app.RequestContext) {
	items, err := h.service.ListExecutions(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h Handler) getExecution(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid job execution id")
		return
	}
	item, err := h.service.GetExecution(ctx, id)
	if err != nil {
		if errors.Is(err, jobpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "job execution not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) streamExecutionEvents(ctx context.Context, c *app.RequestContext) {
	execID, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid job execution id")
		return
	}

	httpx.PrepareStream(c, func(w *httpx.StreamWriter) {
		const (
			minPoll = 200 * time.Millisecond
			maxPoll = 2 * time.Second
		)
		afterID := int64(0)
		wait := minPoll

		for {
			logs, err := h.service.ListLogsAfter(ctx, execID, afterID)
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

			execution, err := h.service.GetExecution(ctx, execID)
			if err == nil && execution != nil &&
				(execution.Status == jobdomain.StatusSucceeded || execution.Status == jobdomain.StatusFailed) {
				_ = w.WriteEvent("state", execution)
				return
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
