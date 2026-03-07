package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/router"
	"octomanger/backend/internal/service"
	"octomanger/backend/internal/task"
	"octomanger/backend/internal/worker/bridge"
	"octomanger/backend/pkg/database"
	"octomanger/backend/pkg/logger"
	"octomanger/backend/pkg/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if mode := strings.TrimSpace(cfg.Server.Mode); mode != "" {
		gin.SetMode(mode)
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

	redisClient := redis.Init(cfg.Redis)
	defer func() {
		_ = redisClient.Close()
	}()

	asynqAddr := cfg.Asynq.EffectiveRedisAddr(cfg.Redis.Addr)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     asynqAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() {
		_ = asynqClient.Close()
	}()

	dispatcher := task.NewProducer(asynqClient)
	pythonBridge := bridge.PythonBridge{
		Binary:  cfg.Python.Bin,
		Script:  cfg.Python.Script,
		Timeout: cfg.Python.Timeout(),
	}

	accountTypeRepo := repository.NewAccountTypeRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	accountSessionRepo := repository.NewAccountSessionRepository(db)
	emailAccountRepo := repository.NewEmailAccountRepository(db)
	jobRepo := repository.NewJobRepository(db)
	jobRunRepo := repository.NewJobRunRepository(db)
	apiKeyRepo := repository.NewApiKeyRepository(db)
	triggerRepo := repository.NewTriggerEndpointRepository(db)
	systemConfigRepo := repository.NewSystemConfigRepository(db)

	accountTypeSvc := service.NewAccountTypeService(accountTypeRepo, cfg.Paths.OctoModuleDir)
	accountSvc := service.NewAccountService(accountRepo, accountTypeRepo, dispatcher, jobRepo)
	emailSvc := service.NewEmailAccountService(emailAccountRepo, service.NewGoEmailBatchRegistrar(), dispatcher, redisClient, jobRepo)
	octoSvc := service.NewOctoModuleService(accountTypeRepo, jobRunRepo, pythonBridge, cfg.Paths.OctoModuleDir, cfg.Python.Bin)
	jobSvc := service.NewJobService(jobRepo, jobRunRepo, accountTypeRepo, dispatcher)
	apiKeySvc := service.NewApiKeyService(apiKeyRepo)
	jobExecutor := service.NewJobExecutor(service.JobExecutorOptions{
		Logger:             log,
		JobRepo:            jobRepo,
		AccountRepo:        accountRepo,
		AccountTypeRepo:    accountTypeRepo,
		JobRunRepo:         jobRunRepo,
		AccountSessionRepo: accountSessionRepo,
		PythonBridge:       pythonBridge,
		ModuleDir:          strings.TrimSpace(cfg.Paths.OctoModuleDir),
	})
	triggerSvc := service.NewTriggerService(triggerRepo, accountTypeRepo, jobRepo, dispatcher, jobExecutor)
	systemConfigSvc := service.NewSystemConfigService(systemConfigRepo)
	systemSvc := service.NewSystemService(db, apiKeySvc)

	services := service.Container{
		AccountType:  accountTypeSvc,
		Account:      accountSvc,
		EmailAccount: emailSvc,
		OctoModule:   octoSvc,
		Job:          jobSvc,
		ApiKey:       apiKeySvc,
		Trigger:      triggerSvc,
		SystemConfig: systemConfigSvc,
		System:       systemSvc,
	}

	engine := router.NewRouter(router.Dependencies{
		Services:   services,
		Logger:     log,
		WebDistDir: cfg.Server.WebDistDir,
	})

	addr := normalizeAddr(cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("api server started", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("api server failed", zap.Error(err))
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("api server shutdown failed", zap.Error(err))
	}
	log.Info("api server stopped")
}

func normalizeAddr(port string) string {
	trimmed := strings.TrimSpace(port)
	if trimmed == "" {
		return ":8080"
	}
	if strings.HasPrefix(trimmed, ":") {
		return trimmed
	}
	return ":" + trimmed
}
