package runtime

import (
	"context"

	"go.uber.org/zap"
)

func syncPluginAccountTypesOnStartup(ctx context.Context, app *App) {
	items, err := app.Plugins.List(ctx)
	if err != nil {
		app.Logger.Warn("list plugins before startup sync failed", zap.Error(err))
		return
	}

	var healthy int
	for _, item := range items {
		if item.Healthy {
			healthy++
		}
	}

	if healthy == 0 {
		app.Logger.Info("plugin account type sync skipped on startup", zap.Int("healthy_plugins", 0))
		return
	}

	if err := app.SyncPluginAccountTypes(ctx); err != nil {
		app.Logger.Warn("plugin account type sync failed", zap.Error(err), zap.Int("healthy_plugins", healthy))
		return
	}

	app.Logger.Info("plugin account types synced", zap.Int("healthy_plugins", healthy))
}
