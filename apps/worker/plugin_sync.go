package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	platformruntime "octomanger/internal/platform/runtime"
)

func startPluginAccountTypeSync(ctx context.Context, application *platformruntime.App, interval time.Duration) {
	if len(application.Config.Plugins.Services) == 0 {
		return
	}

	go syncPluginAccountTypes(ctx, application, interval)
}

func syncPluginAccountTypes(ctx context.Context, application *platformruntime.App, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	synced := make(map[string]struct{})

	trySync := func() bool {
		plugins, err := application.Plugins.List(ctx)
		if err != nil {
			application.Logger.Debug("list plugins before account type sync failed", zap.Error(err))
			return false
		}

		var healthy int
		var pending int
		for _, plugin := range plugins {
			if plugin.Healthy {
				healthy++
				if _, ok := synced[plugin.Manifest.Key]; !ok {
					pending++
				}
			}
		}
		if pending == 0 {
			return len(plugins) > 0 && len(synced) == len(plugins)
		}
		if healthy == 0 {
			return false
		}

		if err := application.SyncPluginAccountTypes(ctx); err != nil {
			application.Logger.Warn("plugin account type sync after launcher startup failed", zap.Error(err))
			return false
		}

		for _, plugin := range plugins {
			if plugin.Healthy {
				synced[plugin.Manifest.Key] = struct{}{}
			}
		}

		application.Logger.Info(
			"plugin account types synced after launcher startup",
			zap.Int("healthy_plugins", healthy),
			zap.Int("synced_plugins", len(synced)),
			zap.Int("registered_plugins", len(plugins)),
		)
		return len(plugins) > 0 && len(synced) == len(plugins)
	}

	if trySync() {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if trySync() {
				return
			}
		}
	}
}
