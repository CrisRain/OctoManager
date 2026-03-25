package runtime

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/logging"
	redisclient "octomanger/internal/platform/redis"
)

type platformResources struct {
	cfg    *config.Config
	logger *zap.Logger
	db     *gorm.DB
	rdb    *redis.Client
}

func bootstrapPlatform(ctx context.Context) (*platformResources, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := logging.New(cfg.Logging)

	db, err := database.Open(cfg.Database)
	if err != nil {
		return nil, err
	}

	rdb, err := redisclient.New(cfg.Redis)
	if err != nil {
		logger.Warn("redis unavailable, continuing without cache", zap.Error(err))
		rdb = nil
	}

	return &platformResources{
		cfg:    &cfg,
		logger: logger,
		db:     db,
		rdb:    rdb,
	}, nil
}
