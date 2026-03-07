package main

import (
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/scheduler"
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

	asynqAddr := cfg.Asynq.EffectiveRedisAddr(cfg.Redis.Addr)
	redisOpt := asynq.RedisClientOpt{
		Addr:     asynqAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	provider := scheduler.NewDBProvider(db)

	mgr, err := asynq.NewPeriodicTaskManager(asynq.PeriodicTaskManagerOpts{
		RedisConnOpt:               redisOpt,
		SyncInterval:               1 * time.Minute, // re-reads DB every minute to pick up new scheduled jobs
		PeriodicTaskConfigProvider: provider,
	})
	if err != nil {
		log.Fatal("failed to create periodic task manager", zap.Error(err))
	}

	log.Info("scheduler started", zap.String("redis", asynqAddr))
	if err := mgr.Run(); err != nil {
		log.Fatal("scheduler failed", zap.Error(err))
	}
}
