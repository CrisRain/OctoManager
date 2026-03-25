package plugins

import (
	"context"

	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
)

// PluginService is the shared interface used by the gRPC-backed plugin runtime.
//
// Callers (jobapp, agentapp, accounts transport, plugin transport) depend on
// this interface rather than a concrete client implementation.
type PluginService interface {
	// Execute runs a plugin action and streams events to onEvent until completion.
	Execute(
		ctx context.Context,
		pluginKey string,
		request plugindomain.ExecutionRequest,
		onEvent func(plugindomain.ExecutionEvent),
	) error

	// List returns all registered plugins.
	List(ctx context.Context) ([]plugindomain.Plugin, error)

	// Get returns a single plugin by key.
	Get(ctx context.Context, key string) (*plugindomain.Plugin, error)

	// SyncAccountTypes calls fn once per plugin with its account-type spec.
	// Used during bootstrap to upsert account types into the database.
	SyncAccountTypes(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error
}
