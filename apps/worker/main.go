package main

import (
	"context"
	platformruntime "octomanger/internal/platform/runtime"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application, err := platformruntime.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	defer application.Close()

	startPluginLauncher(ctx, application)
	startPluginAccountTypeSync(ctx, application, 5*time.Second)
	startAgentSupervisor(ctx, application)
	runJobLoop(ctx, application)
}
