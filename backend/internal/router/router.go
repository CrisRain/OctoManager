package router

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"octomanger/backend/internal/handler"
	"octomanger/backend/internal/middleware"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type Dependencies struct {
	Services   service.Container
	Logger     *zap.Logger
	WebDistDir string
}

func NewRouter(deps Dependencies) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(deps.Logger))

	webDistDir := strings.TrimSpace(deps.WebDistDir)
	if webDistDir != "" {
		if info, err := os.Stat(webDistDir); err != nil || !info.IsDir() {
			if deps.Logger != nil {
				deps.Logger.Warn("web dist dir not found; static serving disabled", zap.String("path", webDistDir), zap.Error(err))
			}
			webDistDir = ""
		}
	}

	healthHandler := handler.NewHealthHandler()
	r.GET("/healthz", healthHandler.Get)

	accountTypeHandler := handler.NewAccountTypeHandler(deps.Services.AccountType)
	accountHandler := handler.NewAccountHandler(deps.Services.Account)
	emailHandler := handler.NewEmailAccountHandler(deps.Services.EmailAccount)
	octoModuleHandler := handler.NewOctoModuleHandler(deps.Services.OctoModule)
	jobHandler := handler.NewJobHandler(deps.Services.Job)
	apiKeyHandler := handler.NewApiKeyHandler(deps.Services.ApiKey)
	triggerHandler := handler.NewTriggerHandlerWithAuth(deps.Services.Trigger, deps.Services.ApiKey)
	systemConfigHandler := handler.NewSystemConfigHandler(deps.Services.SystemConfig)
	systemHandler := handler.NewSystemHandler(deps.Services.System)

	// System routes — no auth guard on status/setup so the UI can bootstrap.
	system := r.Group("/api/v1/system")
	system.GET("/status", systemHandler.Status)
	system.POST("/setup", systemHandler.Setup)
	system.POST("/migrate", middleware.AdminKeyAuth(deps.Services.ApiKey), systemHandler.Migrate)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.ApiKeyAuth(deps.Services.ApiKey))

	accountTypes := v1.Group("/account-types")
	accountTypes.GET("/", accountTypeHandler.List)
	accountTypes.GET("/:key", accountTypeHandler.Get)
	accountTypes.POST("/", accountTypeHandler.Create)
	accountTypes.PATCH("/:key", accountTypeHandler.Patch)
	accountTypes.DELETE("/:key", accountTypeHandler.Delete)

	accounts := v1.Group("/accounts")
	accounts.GET("/", accountHandler.List)
	accounts.POST("/", accountHandler.Create)
	accounts.POST("/batch-patch", accountHandler.BatchPatch)
	accounts.POST("/batch-delete", accountHandler.BatchDelete)
	accounts.GET("/:id", accountHandler.Get)
	accounts.PATCH("/:id", accountHandler.Patch)
	accounts.DELETE("/:id", accountHandler.Delete)

	emailAccounts := v1.Group("/email/accounts")
	emailAccounts.GET("/", emailHandler.List)
	emailAccounts.POST("/", emailHandler.Create)
	emailAccounts.POST("/batch-import-graph", emailHandler.BatchImportGraph)
	emailAccounts.POST("/batch-register", emailHandler.BatchRegister)
	emailAccounts.POST("/batch-delete", emailHandler.BatchDelete)
	emailAccounts.POST("/batch-verify", emailHandler.BatchVerify)
	emailAccounts.POST("/outlook/oauth/authorize-url", emailHandler.BuildOutlookAuthorizeURL)
	emailAccounts.POST("/outlook/oauth/token", emailHandler.ExchangeOutlookCode)
	emailAccounts.POST("/outlook/oauth/refresh", emailHandler.RefreshOutlookToken)
	emailAccounts.POST("/preview/messages/latest", emailHandler.PreviewLatestMessage)
	emailAccounts.POST("/preview/mailboxes", emailHandler.PreviewMailboxes)
	emailAccounts.GET("/:id/mailboxes", emailHandler.ListMailboxes)
	emailAccounts.GET("/:id/messages", emailHandler.ListMessages)
	emailAccounts.GET("/:id/messages/latest", emailHandler.GetLatestMessage)
	emailAccounts.GET("/:id/messages/:messageId", emailHandler.GetMessage)
	emailAccounts.GET("/:id", emailHandler.Get)
	emailAccounts.PATCH("/:id", emailHandler.Patch)
	emailAccounts.POST("/:id", emailHandler.Verify)
	emailAccounts.DELETE("/:id", emailHandler.Delete)

	octoModules := v1.Group("/octo-modules")
	octoModules.GET("/", octoModuleHandler.List)
	octoModules.POST("/sync", octoModuleHandler.Sync)
	octoModules.GET("/:typeKey", octoModuleHandler.Get)
	octoModules.POST("/:typeKey", octoModuleHandler.Action)
	octoModules.GET("/:typeKey/script", octoModuleHandler.GetScript)
	octoModules.PUT("/:typeKey/script", octoModuleHandler.UpdateScript)
	octoModules.GET("/:typeKey/runs", octoModuleHandler.GetRunHistory)
	octoModules.GET("/:typeKey/files", octoModuleHandler.ListFiles)
	octoModules.GET("/:typeKey/files/*filename", octoModuleHandler.GetFile)
	octoModules.PUT("/:typeKey/files/*filename", octoModuleHandler.UpdateFile)
	octoModules.GET("/:typeKey/venv", octoModuleHandler.GetVenvInfo)
	octoModules.POST("/:typeKey/venv/install", octoModuleHandler.InstallDeps)

	jobs := v1.Group("/jobs")
	jobs.POST("/", jobHandler.Create)
	jobs.GET("/", jobHandler.List)
	jobs.GET("/:id", jobHandler.Get)
	jobs.PATCH("/:id", jobHandler.Patch)
	jobs.POST("/:id", jobHandler.Cancel)
	jobs.DELETE("/:id", jobHandler.Delete)

	apiKeys := v1.Group("/api-keys")
	apiKeys.GET("/", apiKeyHandler.List)
	apiKeys.POST("/", apiKeyHandler.Create)
	apiKeys.PATCH("/:id", apiKeyHandler.SetEnabled)
	apiKeys.DELETE("/:id", apiKeyHandler.Delete)

	triggers := v1.Group("/triggers")
	triggers.GET("/", triggerHandler.List)
	triggers.POST("/", triggerHandler.Create)
	triggers.GET("/:id", triggerHandler.Get)
	triggers.PATCH("/:id", triggerHandler.Patch)
	triggers.DELETE("/:id", triggerHandler.Delete)

	r.POST("/webhooks/:slug", triggerHandler.Fire)

	configs := v1.Group("/config")
	configs.GET("/:key", systemConfigHandler.Get)
	configs.PUT("/:key", systemConfigHandler.Set)

	r.NoRoute(func(c *gin.Context) {
		if serveFrontend(c, webDistDir) {
			return
		}
		response.Fail(c, http.StatusNotFound, "not found")
	})

	return r
}

func serveFrontend(c *gin.Context, dist string) bool {
	if dist == "" {
		return false
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}

	reqPath := c.Request.URL.Path
	if strings.HasPrefix(reqPath, "/api/") || strings.HasPrefix(reqPath, "/webhooks/") || reqPath == "/healthz" {
		return false
	}

	cleaned := path.Clean(reqPath)
	relative := strings.TrimPrefix(cleaned, "/")
	if relative == "." || relative == "" {
		relative = "index.html"
	}

	filePath := filepath.Join(dist, filepath.FromSlash(relative))
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		c.File(filePath)
		return true
	}

	indexPath := filepath.Join(dist, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		c.File(indexPath)
		return true
	}

	return false
}
