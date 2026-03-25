package runtime

import (
	"context"

	"go.uber.org/zap"

	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/grpcclient"
	systemapp "octomanger/internal/domains/system/app"
)

func bootstrapPluginService(ctx context.Context, resources *platformResources) (plugins.PluginService, error) {
	timeouts := pluginapp.ExecutionTimeouts{
		Account: resources.cfg.Plugins.Timeout.Account,
		Job:     resources.cfg.Plugins.Timeout.Job,
		Agent:   resources.cfg.Plugins.Timeout.Agent,
	}

	configStore := systemapp.New(resources.db, nil)
	resolvedServices, err := resolvePluginServices(ctx, configStore, resources.cfg.Plugins.Services)
	if err != nil {
		return nil, err
	}
	resources.cfg.Plugins.Services = resolvedServices

	resources.logger.Info("plugin backend: gRPC microservices",
		zap.Int("registered_services", len(resources.cfg.Plugins.Services)),
	)

	registry := grpcclient.NewStaticRegistry(toGRPCServiceMap(resources.cfg.Plugins.Services))
	client := grpcclient.New(registry).WithExecutionTimeouts(timeouts)
	return client.WithSettingsStore(configStore), nil
}
