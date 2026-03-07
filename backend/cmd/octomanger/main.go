// Package main is the unified entry point for OctoManger.
// It runs any combination of the four services in a single process.
//
// Usage:
//
//	go run ./cmd/octomanger                          # all services
//	go run ./cmd/octomanger -services=api,worker     # selected services
//	SERVICES=api,worker go run ./cmd/octomanger      # via env var
//
// Available service names: api, worker, scheduler, daemon
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/backend/config"
	"octomanger/backend/internal/daemon"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/router"
	"octomanger/backend/internal/scheduler"
	"octomanger/backend/internal/service"
	"octomanger/backend/internal/task"
	taskhandler "octomanger/backend/internal/task/handler"
	"octomanger/backend/internal/worker/bridge"
	"octomanger/backend/pkg/database"
	"octomanger/backend/pkg/logger"
	"octomanger/backend/pkg/redis"
)

func main() {
	var servicesFlag string
	flag.StringVar(&servicesFlag, "services", "", "comma-separated services to run: api,worker,scheduler,daemon (default: all)")
	flag.Parse()

	if servicesFlag == "" {
		servicesFlag = os.Getenv("SERVICES")
	}
	enabled := parseServices(servicesFlag)

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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	asynqAddr := cfg.Asynq.EffectiveRedisAddr(cfg.Redis.Addr)
	redisOpt := asynq.RedisClientOpt{
		Addr:     asynqAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	pythonBridge := bridge.PythonBridge{
		Binary:  cfg.Python.Bin,
		Script:  cfg.Python.Script,
		Timeout: cfg.Python.Timeout(),
	}
	moduleDir := strings.TrimSpace(cfg.Paths.OctoModuleDir)

	log.Info("octomanger starting", zap.Strings("services", sortedKeys(enabled)))

	var wg sync.WaitGroup

	if enabled["api"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runAPI(ctx, cfg, db, log, asynqAddr, pythonBridge, moduleDir)
		}()
	}

	if enabled["worker"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWorker(ctx, cfg, db, log, redisOpt, pythonBridge, moduleDir)
		}()
	}

	if enabled["scheduler"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runScheduler(ctx, db, log, redisOpt)
		}()
	}

	if enabled["daemon"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runDaemon(ctx, cfg, db, log, moduleDir)
		}()
	}

	wg.Wait()
	log.Info("octomanger stopped")
}

// ── service runners ────────────────────────────────────────────────────────────

func runAPI(ctx context.Context, cfg config.Config, db *gorm.DB, log *zap.Logger,
	asynqAddr string, pythonBridge bridge.PythonBridge, moduleDir string) {

	redisClient := redis.Init(cfg.Redis)
	defer func() { _ = redisClient.Close() }()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     asynqAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = asynqClient.Close() }()

	dispatcher := task.NewProducer(asynqClient)

	accountTypeRepo := repository.NewAccountTypeRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	accountSessionRepo := repository.NewAccountSessionRepository(db)
	emailAccountRepo := repository.NewEmailAccountRepository(db)
	jobRepo := repository.NewJobRepository(db)
	jobRunRepo := repository.NewJobRunRepository(db)
	apiKeyRepo := repository.NewApiKeyRepository(db)
	triggerRepo := repository.NewTriggerEndpointRepository(db)
	systemConfigRepo := repository.NewSystemConfigRepository(db)

	accountTypeSvc := service.NewAccountTypeService(accountTypeRepo, moduleDir)
	accountSvc := service.NewAccountService(accountRepo, accountTypeRepo, dispatcher, jobRepo)
	emailSvc := service.NewEmailAccountService(emailAccountRepo, service.NewGoEmailBatchRegistrar(), dispatcher, redisClient, jobRepo)
	octoSvc := service.NewOctoModuleService(accountTypeRepo, jobRunRepo, pythonBridge, moduleDir, cfg.Python.Bin)
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
		ModuleDir:          moduleDir,
	})
	triggerSvc := service.NewTriggerService(triggerRepo, accountTypeRepo, jobRepo, dispatcher, jobExecutor)
	systemConfigSvc := service.NewSystemConfigService(systemConfigRepo)
	systemSvc := service.NewSystemService(db, apiKeySvc)

	engine := router.NewRouter(router.Dependencies{
		Services: service.Container{
			AccountType:  accountTypeSvc,
			Account:      accountSvc,
			EmailAccount: emailSvc,
			OctoModule:   octoSvc,
			Job:          jobSvc,
			ApiKey:       apiKeySvc,
			Trigger:      triggerSvc,
			SystemConfig: systemConfigSvc,
			System:       systemSvc,
		},
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

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("api server shutdown failed", zap.Error(err))
	}
	log.Info("api server stopped")
}

func runWorker(ctx context.Context, cfg config.Config, db *gorm.DB, log *zap.Logger,
	redisOpt asynq.RedisClientOpt, pythonBridge bridge.PythonBridge, moduleDir string) {

	concurrency := cfg.Asynq.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: concurrency,
		Queues:      map[string]int{"critical": 6, "default": 3, "low": 1},
	})

	jobHandler := taskhandler.NewJobHandler(taskhandler.JobHandlerOptions{
		Logger:             log,
		JobRepo:            repository.NewJobRepository(db),
		AccountRepo:        repository.NewAccountRepository(db),
		AccountTypeRepo:    repository.NewAccountTypeRepository(db),
		JobRunRepo:         repository.NewJobRunRepository(db),
		AccountSessionRepo: repository.NewAccountSessionRepository(db),
		PythonBridge:       pythonBridge,
		ModuleDir:          moduleDir,
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

	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()

	log.Info("worker started")
	if err := srv.Run(mux); err != nil {
		log.Error("worker stopped", zap.Error(err))
	}
}

func runScheduler(ctx context.Context, db *gorm.DB, log *zap.Logger, redisOpt asynq.RedisClientOpt) {
	provider := scheduler.NewDBProvider(db)
	mgr, err := asynq.NewPeriodicTaskManager(asynq.PeriodicTaskManagerOpts{
		RedisConnOpt:               redisOpt,
		SyncInterval:               1 * time.Minute,
		PeriodicTaskConfigProvider: provider,
	})
	if err != nil {
		log.Fatal("failed to create scheduler", zap.Error(err))
	}

	go func() {
		<-ctx.Done()
		mgr.Shutdown()
	}()

	log.Info("scheduler started")
	if err := mgr.Run(); err != nil {
		log.Error("scheduler stopped", zap.Error(err))
	}
}

func runDaemon(ctx context.Context, cfg config.Config, db *gorm.DB, log *zap.Logger, moduleDir string) {
	mgr := daemon.NewManager(db, daemon.Config{
		PythonBin: cfg.Python.Bin,
		ModuleDir: moduleDir,
	}, log)

	log.Info("daemon manager started")
	if err := mgr.Run(ctx); err != nil {
		log.Error("daemon manager stopped", zap.Error(err))
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

var allServices = []string{"api", "worker", "scheduler", "daemon"}

func parseServices(s string) map[string]bool {
	all := map[string]bool{"api": true, "worker": true, "scheduler": true, "daemon": true}
	s = strings.TrimSpace(s)
	if s == "" || s == "all" {
		return all
	}
	enabled := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if all[name] {
			enabled[name] = true
		}
	}
	if len(enabled) == 0 {
		return all
	}
	return enabled
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
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
