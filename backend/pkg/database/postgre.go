package database

import (
	"errors"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"octomanger/backend/config"
)

func Init(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := strings.TrimSpace(cfg.ConnectionString())
	if dsn == "" {
		return nil, errors.New("database dsn is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if lifetime := cfg.ConnMaxLifetime(); lifetime > 0 {
		sqlDB.SetConnMaxLifetime(lifetime)
	}
	if idle := cfg.ConnMaxIdleTime(); idle > 0 {
		sqlDB.SetConnMaxIdleTime(idle)
	}

	return db, nil
}
