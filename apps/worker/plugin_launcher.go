package main

import (
	"context"

	"go.uber.org/zap"

	"octomanger/internal/domains/plugins/grpclauncher"
	platformconfig "octomanger/internal/platform/config"
	platformruntime "octomanger/internal/platform/runtime"
)

func startPluginLauncher(ctx context.Context, application *platformruntime.App) {
	launches := grpclauncher.Discover(application.Config.Plugins.ModulesDir, toAddressMap(application.Config.Plugins.Services))
	go func() {
		manager := grpclauncher.New(
			application.Logger.Named("plugin-launcher"),
			application.Config.Plugins.PythonBin,
			application.Config.Plugins.SDKDir,
			launches,
		)
		if err := manager.Run(ctx); err != nil && ctx.Err() == nil {
			application.Logger.Error("plugin launcher stopped unexpectedly", zap.Error(err))
		}
	}()
}

func toAddressMap(services map[string]platformconfig.PluginServiceEntry) map[string]string {
	result := make(map[string]string, len(services))
	for key, service := range services {
		result[key] = service.Address
	}
	return result
}
