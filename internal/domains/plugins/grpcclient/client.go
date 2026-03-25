// Package grpcclient implements the plugins.PluginService interface by
// communicating with persistent plugin gRPC microservices instead of spawning
// per-request Python subprocesses.
//
// Each plugin runs as a long-lived gRPC server (started externally or via
// docker-compose). The client maintains one connection per plugin key and
// reconnects transparently using gRPC's built-in backoff.
package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
	pluginv1 "octomanger/internal/gen/plugin/v1"
	"octomanger/internal/platform/grpccodec"
)

// Client is a gRPC-backed implementation of plugins.PluginService.
// It is safe for concurrent use.
type Client struct {
	registry      PluginRegistry
	settingsStore pluginapp.SettingsStore // may be nil
	timeouts      pluginapp.ExecutionTimeouts

	mu    sync.RWMutex
	conns map[string]*grpc.ClientConn // normalised key → live connection
}

const metadataRPCTimeout = 2 * time.Second

// New creates a Client. Call WithSettingsStore and WithExecutionTimeouts to
// configure optional behaviour before the first Execute call.
func New(registry PluginRegistry) *Client {
	return &Client{
		registry: registry,
		timeouts: defaultTimeouts(),
		conns:    make(map[string]*grpc.ClientConn),
	}
}

// WithSettingsStore attaches a settings store used to inject per-plugin
// settings into every ExecuteRequest context field.
func (c *Client) WithSettingsStore(store pluginapp.SettingsStore) *Client {
	c.settingsStore = store
	return c
}

// WithExecutionTimeouts overrides the per-mode execution timeouts.
func (c *Client) WithExecutionTimeouts(t pluginapp.ExecutionTimeouts) *Client {
	c.timeouts = t
	return c
}

// ── PluginService interface ──────────────────────────────────────────────────

// Execute calls the plugin's Execute RPC and streams events back via onEvent.
// It mirrors the semantics of the subprocess backend: events arrive in order,
// the call returns when the stream closes (plugin finished) or context expires.
func (c *Client) Execute(
	ctx context.Context,
	pluginKey string,
	request plugindomain.ExecutionRequest,
	onEvent func(plugindomain.ExecutionEvent),
) error {
	stub, err := c.stub(pluginKey)
	if err != nil {
		return err
	}

	request, err = c.injectSettings(ctx, pluginKey, request)
	if err != nil {
		return err
	}

	execCtx := ctx
	timeout := c.timeoutFor(request)
	var cancel context.CancelFunc = func() {}
	if timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	inputJSON, err := json.Marshal(request.Input)
	if err != nil {
		return fmt.Errorf("grpcclient: marshal request.input: %w", err)
	}
	ctxJSON, err := json.Marshal(request.Context)
	if err != nil {
		return fmt.Errorf("grpcclient: marshal request.context: %w", err)
	}

	stream, err := stub.Execute(execCtx, &pluginv1.ExecuteRequest{
		Mode:    request.Mode,
		Action:  request.Action,
		Input:   inputJSON,
		Context: ctxJSON,
	})
	if err != nil {
		return fmt.Errorf("grpcclient: open execute stream for %q: %w", pluginKey, err)
	}

	var receivedError bool
	for {
		ev, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if timeout > 0 && execCtx.Err() == context.DeadlineExceeded {
				msg := fmt.Sprintf("plugin execution timed out after %s", timeout)
				if onEvent != nil {
					onEvent(plugindomain.ExecutionEvent{Type: "error", Message: msg, Error: "TIMEOUT"})
				}
				return fmt.Errorf("%s", msg)
			}
			if receivedError {
				// Plugin already communicated the failure via an error event.
				return nil
			}
			return fmt.Errorf("grpcclient: receive event from %q: %w", pluginKey, err)
		}

		event := protoToEvent(ev)
		if event.Type == "error" {
			receivedError = true
		}
		if onEvent != nil {
			onEvent(event)
		}
	}
	return nil
}

// List returns lightweight plugin records for all services registered in the registry.
// It calls GetManifest on each plugin to populate the manifest fields.
func (c *Client) List(ctx context.Context) ([]plugindomain.Plugin, error) {
	keys := c.registry.Keys()
	plugins := make([]plugindomain.Plugin, 0, len(keys))
	for _, key := range keys {
		p, err := c.Get(ctx, key)
		if err != nil || p == nil {
			// Unhealthy or not yet up — include a minimal record.
			plugins = append(plugins, plugindomain.Plugin{
				Manifest: plugindomain.Manifest{Key: key},
				Healthy:  false,
			})
			continue
		}
		plugins = append(plugins, *p)
	}
	return plugins, nil
}

// Get returns a single plugin by key, fetching its manifest via GetManifest RPC.
func (c *Client) Get(ctx context.Context, key string) (*plugindomain.Plugin, error) {
	stub, err := c.stub(key)
	if err != nil {
		return &plugindomain.Plugin{Manifest: placeholderManifest(key), Healthy: false}, nil
	}

	rpcCtx, cancel := metadataContext(ctx)
	resp, err := stub.GetManifest(rpcCtx, &pluginv1.GetManifestRequest{})
	cancel()
	if err != nil {
		return &plugindomain.Plugin{Manifest: placeholderManifest(key), Healthy: false}, nil
	}

	manifest, err := decodeManifest(resp.Manifest, key)
	if err != nil {
		return nil, fmt.Errorf("grpcclient: parse manifest for %q: %w", key, err)
	}

	return &plugindomain.Plugin{
		Manifest: manifest,
		Healthy:  true,
	}, nil
}

// SyncAccountTypes calls fn for every plugin in the registry, fetching manifests
// via GetManifest RPC. This replaces the filesystem account_type.{key}.json reads.
func (c *Client) SyncAccountTypes(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
	keys := c.registry.Keys()
	for _, key := range keys {
		stub, err := c.stub(key)
		if err != nil {
			continue // plugin not reachable — skip silently like the FS backend
		}

		rpcCtx, cancel := metadataContext(ctx)
		resp, err := stub.GetManifest(rpcCtx, &pluginv1.GetManifestRequest{})
		cancel()
		if err != nil {
			continue
		}

		var spec pluginapp.AccountTypeSpec
		if err := json.Unmarshal(resp.Manifest, &spec); err != nil {
			return fmt.Errorf("grpcclient: parse manifest for %q: %w", key, err)
		}
		if spec.Key == "" {
			spec.Key = key
		}
		if spec.Category == "" {
			spec.Category = "generic"
		}
		if err := fn(ctx, spec); err != nil {
			return fmt.Errorf("grpcclient: sync account type %q: %w", spec.Key, err)
		}
	}
	return nil
}

// HealthCheck probes the given plugin and returns whether it is alive.
func (c *Client) HealthCheck(ctx context.Context, pluginKey string) (bool, string) {
	stub, err := c.stub(pluginKey)
	if err != nil {
		return false, ""
	}
	rpcCtx, cancel := metadataContext(ctx)
	defer cancel()
	resp, err := stub.HealthCheck(rpcCtx, &pluginv1.HealthCheckRequest{})
	if err != nil {
		return false, ""
	}
	return resp.Healthy, resp.Version
}

// Close shuts down all open gRPC connections.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, conn := range c.conns {
		_ = conn.Close()
		delete(c.conns, key)
	}
}

// ── internal helpers ────────────────────────────────────────────────────────

// stub returns (or lazily creates) the gRPC client stub for a plugin key.
func (c *Client) stub(pluginKey string) (pluginv1.PluginServiceClient, error) {
	conn, err := c.conn(pluginKey)
	if err != nil {
		return nil, err
	}
	return pluginv1.NewPluginServiceClient(conn), nil
}

// conn returns (or lazily creates) the gRPC ClientConn for a plugin key.
// Connections are kept alive and reused across calls; gRPC handles reconnection.
func (c *Client) conn(pluginKey string) (*grpc.ClientConn, error) {
	nk := normaliseKey(pluginKey)

	// Fast path: connection already exists and is not shutdown.
	c.mu.RLock()
	conn, ok := c.conns[nk]
	c.mu.RUnlock()
	if ok && conn.GetState() != connectivity.Shutdown {
		return conn, nil
	}

	// Slow path: create a new connection.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check after acquiring write lock.
	if conn, ok = c.conns[nk]; ok && conn.GetState() != connectivity.Shutdown {
		return conn, nil
	}

	addr, err := c.registry.Address(pluginKey)
	if err != nil {
		return nil, err
	}

	newConn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Force JSON codec so plugin services don't need grpcio-tools to
		// generate binary protobuf stubs. Both sides use plain JSON payloads.
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpccodec.Codec())),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("grpcclient: dial plugin %q at %q: %w", pluginKey, addr, err)
	}

	if conn != nil {
		_ = conn.Close() // close stale shutdown connection
	}
	c.conns[nk] = newConn
	return newConn, nil
}

func (c *Client) injectSettings(
	ctx context.Context,
	pluginKey string,
	request plugindomain.ExecutionRequest,
) (plugindomain.ExecutionRequest, error) {
	if c.settingsStore == nil {
		return request, nil
	}
	raw, err := c.settingsStore.GetConfig(ctx, settingsKey(pluginKey))
	if err != nil {
		return request, fmt.Errorf("grpcclient: load settings for %q: %w", pluginKey, err)
	}
	settings := map[string]any{}
	if len(raw) > 0 && strings.TrimSpace(string(raw)) != "" && strings.TrimSpace(string(raw)) != "null" {
		if err := json.Unmarshal(raw, &settings); err != nil {
			return request, fmt.Errorf("grpcclient: decode settings for %q: %w", pluginKey, err)
		}
	}
	if request.Context == nil {
		request.Context = map[string]any{}
	}
	request.Context["settings"] = settings
	return request, nil
}

func (c *Client) timeoutFor(req plugindomain.ExecutionRequest) time.Duration {
	if src, _ := req.Context["source"].(string); strings.TrimSpace(src) == "account-execute" {
		return c.timeouts.Account
	}
	switch strings.ToLower(strings.TrimSpace(req.Mode)) {
	case "agent":
		return c.timeouts.Agent
	case "job":
		return c.timeouts.Job
	case "account":
		return c.timeouts.Account
	default:
		return 0
	}
}

// protoToEvent converts a protobuf ExecuteEvent to the domain ExecutionEvent.
func protoToEvent(ev *pluginv1.ExecuteEvent) plugindomain.ExecutionEvent {
	event := plugindomain.ExecutionEvent{
		Type:     ev.Type,
		Message:  ev.Message,
		Progress: int(ev.Progress),
		Error:    ev.Error,
	}
	if len(ev.Data) > 0 {
		var data map[string]any
		if err := json.Unmarshal(ev.Data, &data); err == nil {
			event.Data = data
		}
	}
	if event.Type == "error" && event.Message == "" {
		event.Message = event.Error
	}
	return event
}

func settingsKey(pluginKey string) string {
	return "plugin_settings:" + pluginKey
}

func defaultTimeouts() pluginapp.ExecutionTimeouts {
	return pluginapp.ExecutionTimeouts{
		Account: 60 * time.Second,
		Job:     10 * time.Minute,
		Agent:   0,
	}
}

type manifestPayload struct {
	Key          string            `json:"key"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Runtime      string            `json:"runtime"`
	Entrypoint   string            `json:"entrypoint"`
	Schema       map[string]any    `json:"schema"`
	Settings     []map[string]any  `json:"settings"`
	UI           map[string]any    `json:"ui"`
	Metadata     map[string]string `json:"metadata"`
	Capabilities any               `json:"capabilities"`
}

type manifestCapabilities struct {
	Actions []manifestAction `json:"actions"`
}

type manifestAction struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Modes       []string `json:"modes"`
}

func decodeManifest(raw []byte, fallbackKey string) (plugindomain.Manifest, error) {
	var payload manifestPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return plugindomain.Manifest{}, err
	}

	manifest := plugindomain.Manifest{
		Key:           strings.TrimSpace(payload.Key),
		Name:          strings.TrimSpace(payload.Name),
		Version:       strings.TrimSpace(payload.Version),
		Description:   strings.TrimSpace(payload.Description),
		Runtime:       strings.TrimSpace(payload.Runtime),
		Entrypoint:    strings.TrimSpace(payload.Entrypoint),
		AccountSchema: payload.Schema,
		Settings:      payload.Settings,
		UI:            payload.UI,
		Metadata:      payload.Metadata,
	}
	if manifest.Key == "" {
		manifest.Key = strings.TrimSpace(fallbackKey)
	}
	if manifest.Name == "" {
		manifest.Name = manifest.Key
	}

	actions, capabilities := decodeCapabilities(payload.Capabilities)
	manifest.Actions = actions
	manifest.Capabilities = capabilities

	return manifest, nil
}

func decodeCapabilities(raw any) ([]plugindomain.ManifestAction, []string) {
	if raw == nil {
		return nil, nil
	}

	switch value := raw.(type) {
	case []any:
		capabilities := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				capabilities = append(capabilities, strings.TrimSpace(text))
			}
		}
		return nil, capabilities
	case map[string]any:
		actions := decodeActionSlice(value["actions"])
		capabilities := make([]string, 0, len(value))
		for key := range value {
			if strings.EqualFold(key, "actions") {
				continue
			}
			capabilities = append(capabilities, key)
		}
		return actions, capabilities
	default:
		return nil, nil
	}
}

func decodeActionSlice(raw any) []plugindomain.ManifestAction {
	if raw == nil {
		return nil
	}

	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var payload []manifestAction
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return nil
	}

	actions := make([]plugindomain.ManifestAction, 0, len(payload))
	for _, item := range payload {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = key
		}
		actions = append(actions, plugindomain.ManifestAction{
			Key:         key,
			Name:        name,
			Description: strings.TrimSpace(item.Description),
			Modes:       item.Modes,
		})
	}
	return actions
}

func placeholderManifest(key string) plugindomain.Manifest {
	key = strings.TrimSpace(key)
	return plugindomain.Manifest{
		Key:  key,
		Name: key,
	}
}

func metadataContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, metadataRPCTimeout)
}
