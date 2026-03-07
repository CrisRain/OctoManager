package main

import (
	"strings"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/service"
	"octomanger/backend/internal/task"
	taskhandler "octomanger/backend/internal/task/handler"
	"octomanger/backend/internal/worker/bridge"
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
	defer func() {
		_ = log.Sync()
	}()

	db, err := database.Init(cfg.Database)
	if err != nil {
		log.Fatal("failed to init database", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to init database connection", zap.Error(err))
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	if cfg.Database.AutoMigrate {
		report, err := database.Migrate(db, database.MigrateOptions{DropLegacy: cfg.Database.Reset})
		if err != nil {
			log.Fatal("database migration failed", zap.Error(err))
		}
		if len(report.DroppedTables) > 0 {
			log.Warn("legacy tables dropped", zap.Strings("tables", report.DroppedTables))
		}
		if len(report.DroppedColumns) > 0 {
			log.Warn("legacy columns dropped", zap.Strings("columns", report.DroppedColumns))
		}
	}

	asynqAddr := cfg.Asynq.EffectiveRedisAddr(cfg.Redis.Addr)
	concurrency := cfg.Asynq.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     asynqAddr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	pythonBridge := bridge.PythonBridge{
		Binary:  cfg.Python.Bin,
		Script:  cfg.Python.Script,
		Timeout: cfg.Python.Timeout(),
	}

	jobHandler := taskhandler.NewJobHandler(taskhandler.JobHandlerOptions{
		Logger:             log,
		JobRepo:            repository.NewJobRepository(db),
		AccountRepo:        repository.NewAccountRepository(db),
		AccountTypeRepo:    repository.NewAccountTypeRepository(db),
		JobRunRepo:         repository.NewJobRunRepository(db),
		AccountSessionRepo: repository.NewAccountSessionRepository(db),
		PythonBridge:       pythonBridge,
		ModuleDir:          strings.TrimSpace(cfg.Paths.OctoModuleDir),
	})
	batchHandler := taskhandler.NewBatchHandler(taskhandler.BatchHandlerOptions{
		Logger:           log,
		AccountRepo:      repository.NewAccountRepository(db),
		EmailAccountRepo: repository.NewEmailAccountRepository(db),
		JobRepo:          repository.NewJobRepository(db),
		JobRunRepo:       repository.NewJobRunRepository(db),
		BatchRegistrar:   service.NewGoEmailBatchRegistrar(),
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeDispatchJob, jobHandler.ProcessDispatchJob)
	mux.HandleFunc(task.TypeBatchAccountPatch, batchHandler.ProcessAccountBatchPatch)
	mux.HandleFunc(task.TypeBatchAccountDelete, batchHandler.ProcessAccountBatchDelete)
	mux.HandleFunc(task.TypeBatchEmailDelete, batchHandler.ProcessEmailBatchDelete)
	mux.HandleFunc(task.TypeBatchEmailVerify, batchHandler.ProcessEmailBatchVerify)
	mux.HandleFunc(task.TypeBatchEmailRegister, batchHandler.ProcessEmailBatchRegister)
	mux.HandleFunc(task.TypeBatchEmailImportGraph, batchHandler.ProcessEmailBatchImportGraph)

	log.Info("worker started", zap.String("redis", asynqAddr))
	if err := srv.Run(mux); err != nil {
		log.Fatal("worker failed", zap.Error(err))
	}
}
