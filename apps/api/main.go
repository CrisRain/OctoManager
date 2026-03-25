package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.uber.org/zap"

	accounttypestransport "octomanger/internal/domains/account-types/transport"
	accountstransport "octomanger/internal/domains/accounts/transport"
	agenttransport "octomanger/internal/domains/agents/transport"
	emailtransport "octomanger/internal/domains/email/transport"
	jobtransport "octomanger/internal/domains/jobs/transport"
	plugintransport "octomanger/internal/domains/plugins/transport"
	systemtransport "octomanger/internal/domains/system/transport"
	triggertransport "octomanger/internal/domains/triggers/transport"
	platformruntime "octomanger/internal/platform/runtime"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application, err := platformruntime.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	defer application.Close()

	h := server.New(
		server.WithHostPorts(application.Config.Server.APIAddr),
		server.WithReadTimeout(application.Config.Server.ReadTimeout),
		server.WithIdleTimeout(application.Config.Server.IdleTimeout),
		server.WithExitWaitTime(5*time.Second),
	)

	root := h.Group("/")
	v2 := h.Group("/api/v2")

	// System: /healthz on root, /api/v2/system/* and /api/v2/config/* on v2
	systemtransport.NewHandler(application.Config.Auth.AdminKey, application.System).Register(root, v2)

	// Public domain routes (no blanket auth — each handler applies guard per-route)
	plugintransport.NewHandler(application.Config.Auth.AdminKey, application.Plugins, application.AccountTypes, application.System).Register(v2)
	jobtransport.NewHandler(application.Config.Auth.AdminKey, application.Jobs).Register(v2)
	agenttransport.NewHandler(application.Config.Auth.AdminKey, application.Agents).Register(v2)
	accounttypestransport.NewHandler(application.Config.Auth.AdminKey, application.AccountTypes).Register(v2)
	accountstransport.NewHandler(application.Config.Auth.AdminKey, application.Accounts, application.Plugins).Register(v2)
	emailtransport.NewHandler(application.Config.Auth.AdminKey, application.Email).Register(v2)
	triggertransport.NewHandler(application.Config.Auth.AdminKey, application.Triggers).Register(v2, root)

	// Serve built frontend (SPA fallback)
	distDir := application.Config.Server.WebDistDir
	if distDir != "" {
		h.NoRoute(func(reqCtx context.Context, c *app.RequestContext) {
			file, ok := resolveStaticFile(
				distDir,
				string(c.Method()),
				string(c.Path()),
				string(c.GetHeader("Accept")),
				string(c.GetHeader("Accept-Encoding")),
			)
			if ok {
				applyStaticFileHeaders(c, file)
				app.ServeFileUncompressed(c, file.diskPath)
				return
			}

			c.JSON(consts.StatusNotFound, map[string]any{"error": "not found"})
		})
	}

	go func() {
		<-ctx.Done()
		application.Logger.Info("shutting down api server")
		h.Shutdown(context.Background()) //nolint:errcheck
	}()

	application.Logger.Info("api server starting", zap.String("addr", application.Config.Server.APIAddr))
	h.Spin()
}
