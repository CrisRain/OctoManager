package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/daemon"
	"octomanger/backend/pkg/database"
	"octomanger/backend/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.Init(cfg.Logging)
	if err != nil {
		fallback, _ := zap.NewProduction()
		fallback.Fatal("failed to init logger", zap.Error(err))
	}
	defer func() { _ = log.Sync() }()

	db, err := database.Init(cfg.Database)
	if err != nil {
		log.Fatal("failed to init database", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB", zap.Error(err))
	}
	defer func() { _ = sqlDB.Close() }()

	mgr := daemon.NewManager(db, daemon.Config{
		PythonBin: cfg.Python.Bin,
		ModuleDir: strings.TrimSpace(cfg.Paths.OctoModuleDir),
	}, log)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Info("daemon manager started")
	if err := mgr.Run(ctx); err != nil {
		log.Fatal("daemon manager failed", zap.Error(err))
	}
	log.Info("daemon manager stopped")
}
