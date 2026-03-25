package main

import (
	"context"

	"go.uber.org/zap"

	platformruntime "octomanger/internal/platform/runtime"
)

func startAgentSupervisor(ctx context.Context, application *platformruntime.App) {
	go func() {
		if err := application.Agents.RunSupervisor(ctx, application.Config.Worker.AgentScanInterval); err != nil && ctx.Err() == nil {
			application.Logger.Error("agent supervisor stopped unexpectedly", zap.Error(err))
		}
	}()
}
