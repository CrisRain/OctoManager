package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	platformruntime "octomanger/internal/platform/runtime"
)

func runJobLoop(ctx context.Context, application *platformruntime.App) {
	ticker := time.NewTicker(application.Config.Worker.PollInterval)
	defer ticker.Stop()

	application.Logger.Info("worker started", zap.String("worker_id", application.Config.Worker.ID))

	for {
		runWorkerTick(ctx, application)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func runWorkerTick(ctx context.Context, application *platformruntime.App) {
	if _, err := application.Jobs.TickSchedules(ctx, application.Config.Worker.SchedulePollLimit); err != nil {
		application.Logger.Error("tick schedules failed", zap.Error(err))
	}

	processQueuedExecutions(ctx, application)
}

func processQueuedExecutions(ctx context.Context, application *platformruntime.App) {
	for i := 0; i < application.Config.Worker.ExecutionPollLimit; i++ {
		processed, err := application.Jobs.ProcessNextExecution(ctx)
		if err != nil {
			application.Logger.Error("process execution failed", zap.Error(err))
			return
		}
		if !processed {
			return
		}
	}
}
