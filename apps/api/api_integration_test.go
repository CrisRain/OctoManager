package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	accounttypestransport "octomanger/internal/domains/account-types/transport"
	accountapp "octomanger/internal/domains/accounts/app"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	accountstransport "octomanger/internal/domains/accounts/transport"
	agentapp "octomanger/internal/domains/agents/app"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	agenttransport "octomanger/internal/domains/agents/transport"
	emailapp "octomanger/internal/domains/email/app"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	emailtransport "octomanger/internal/domains/email/transport"
	jobapp "octomanger/internal/domains/jobs/app"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	jobtransport "octomanger/internal/domains/jobs/transport"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/infra/fsrepo"
	plugintransport "octomanger/internal/domains/plugins/transport"
	systemapp "octomanger/internal/domains/system/app"
	systemtransport "octomanger/internal/domains/system/transport"
	triggerapp "octomanger/internal/domains/triggers/app"
	triggerpostgres "octomanger/internal/domains/triggers/infra/postgres"
	triggertransport "octomanger/internal/domains/triggers/transport"
	platformconfig "octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/logging"
)

const testAdminKey = "test-admin-key"

type testAPI struct {
	baseURL string
	jobs    jobapp.Service
	db      *gorm.DB
}

func TestAPISmoke_TriggerToWorkerExecution(t *testing.T) {
	requirePython(t)
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	fakeServerURL := startFakeServer(t, ctx, rootDir)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	definition := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "smoke-job",
		"name":       "Smoke Job",
		"plugin_key": "octo_demo",
		"action":     "LIST_TASKS",
		"input": map[string]any{
			"account": map[string]any{
				"spec": map[string]any{
					"username": "testuser",
					"api_key":  "demo_testkey_12345678",
					"base_url": fakeServerURL,
				},
			},
			"params": map[string]any{
				"page": "1",
			},
		},
	}, adminHeaders(testAdminKey))

	definitionID := int64(definition["id"].(float64))

	triggerCreate := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/triggers", map[string]any{
		"key":               "smoke-trigger",
		"name":              "Smoke Trigger",
		"job_definition_id": definitionID,
		"mode":              "async",
		"default_input": map[string]any{
			"params": map[string]any{
				"page": "1",
			},
		},
		"enabled": true,
	}, adminHeaders(testAdminKey))

	deliveryToken := triggerCreate["delivery_token"].(string)
	fireResponse := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/webhooks/smoke-trigger", map[string]any{
		"params": map[string]any{
			"page": "1",
		},
	}, map[string]string{
		"X-Trigger-Token": deliveryToken,
	})
	if fireResponse["execution_id"] == nil {
		t.Fatalf("expected execution_id in trigger fire response")
	}
	executionID := int64(fireResponse["execution_id"].(float64))

	processed, err := api.jobs.ProcessNextExecution(ctx)
	if err != nil {
		t.Fatalf("process next execution: %v", err)
	}
	if !processed {
		t.Fatalf("expected queued execution to be processed")
	}

	executionDetails := getJSON[map[string]any](t, fmt.Sprintf("%s/api/v2/job-executions/%d", api.baseURL, executionID))
	if executionDetails["status"] != "succeeded" {
		t.Fatalf("expected succeeded execution, got %v", executionDetails["status"])
	}

	var logCount int64
	if err := api.db.WithContext(ctx).Table("job_logs").Where("job_execution_id = ?", executionID).Count(&logCount).Error; err != nil {
		t.Fatalf("count job logs: %v", err)
	}
	if logCount == 0 {
		t.Fatalf("expected imported plugin events to be stored")
	}
}

func TestAPISystemConfigRequiresAdminKey(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	putJSONExpectStatus(t, api.baseURL+"/api/v2/config/app.name", map[string]any{
		"value": "OctoManger Test",
	}, nil, http.StatusUnauthorized)

	updated := putJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/config/app.name", map[string]any{
		"value": "OctoManger Test",
	}, adminHeaders(testAdminKey))
	if updated["key"] != "app.name" {
		t.Fatalf("expected app.name key, got %v", updated["key"])
	}

	value := getJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/config/app.name", adminHeaders(testAdminKey))
	if value["value"] != "OctoManger Test" {
		t.Fatalf("expected updated config value, got %v", value["value"])
	}
}

func TestAPIJobDefinitionPatchUpdatesPluginAndAction(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	definition := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "patch-job",
		"name":       "Patch Job",
		"plugin_key": "octo_demo",
		"action":     "LIST_TASKS",
		"input": map[string]any{
			"params": map[string]any{
				"page": "1",
			},
		},
	}, adminHeaders(testAdminKey))

	definitionID := int64(definition["id"].(float64))

	patched := requestJSON[map[string]any](t, http.MethodPatch, fmt.Sprintf("%s/api/v2/job-definitions/%d", api.baseURL, definitionID), map[string]any{
		"name":       "Patch Job Updated",
		"plugin_key": "octo_demo",
		"action":     "CREATE_TASK",
		"input": map[string]any{
			"params": map[string]any{
				"title": "Created from patch",
			},
		},
	}, adminHeaders(testAdminKey), http.StatusOK)

	if patched["name"] != "Patch Job Updated" {
		t.Fatalf("expected updated job name, got %v", patched["name"])
	}
	if patched["plugin_key"] != "octo_demo" {
		t.Fatalf("expected updated plugin_key, got %v", patched["plugin_key"])
	}
	if patched["action"] != "CREATE_TASK" {
		t.Fatalf("expected updated action, got %v", patched["action"])
	}
}

func TestAPITriggerPatchUpdatesJobAndDefaultInput(t *testing.T) {
	ctx, rootDir, db := setupIntegrationPrereqs(t)
	api := startTestAPI(t, ctx, rootDir, db, testAdminKey)

	jobA := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "trigger-job-a",
		"name":       "Trigger Job A",
		"plugin_key": "octo_demo",
		"action":     "LIST_TASKS",
		"input":      map[string]any{},
	}, adminHeaders(testAdminKey))
	jobB := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/job-definitions", map[string]any{
		"key":        "trigger-job-b",
		"name":       "Trigger Job B",
		"plugin_key": "octo_demo",
		"action":     "CREATE_TASK",
		"input":      map[string]any{},
	}, adminHeaders(testAdminKey))

	trigger := postJSONWithHeaders[map[string]any](t, api.baseURL+"/api/v2/triggers", map[string]any{
		"key":               "patch-trigger",
		"name":              "Patch Trigger",
		"job_definition_id": int64(jobA["id"].(float64)),
		"mode":              "async",
		"default_input": map[string]any{
			"source": "before",
		},
		"enabled": true,
	}, adminHeaders(testAdminKey))

	triggerID := int64(trigger["trigger"].(map[string]any)["id"].(float64))
	jobBID := int64(jobB["id"].(float64))

	patched := requestJSON[map[string]any](t, http.MethodPatch, fmt.Sprintf("%s/api/v2/triggers/%d", api.baseURL, triggerID), map[string]any{
		"name":              "Patch Trigger Updated",
		"job_definition_id": jobBID,
		"mode":              "sync",
		"default_input": map[string]any{
			"source": "after",
			"params": map[string]any{
				"title": "Created from trigger patch",
			},
		},
		"enabled": false,
	}, adminHeaders(testAdminKey), http.StatusOK)

	if patched["name"] != "Patch Trigger Updated" {
		t.Fatalf("expected updated trigger name, got %v", patched["name"])
	}
	if int64(patched["job_definition_id"].(float64)) != jobBID {
		t.Fatalf("expected updated job_definition_id %d, got %v", jobBID, patched["job_definition_id"])
	}
	if patched["mode"] != "sync" {
		t.Fatalf("expected updated trigger mode, got %v", patched["mode"])
	}
	if patched["enabled"] != false {
		t.Fatalf("expected updated enabled flag false, got %v", patched["enabled"])
	}

	defaultInput, ok := patched["default_input"].(map[string]any)
	if !ok {
		t.Fatalf("expected default_input map, got %T", patched["default_input"])
	}
	if defaultInput["source"] != "after" {
		t.Fatalf("expected updated default_input source, got %v", defaultInput["source"])
	}
}

func postJSON[T any](t *testing.T, url string, payload any) T {
	t.Helper()
	return postJSONWithHeaders[T](t, url, payload, nil)
}

func postJSONWithHeaders[T any](t *testing.T, url string, payload any, headers map[string]string) T {
	t.Helper()
	return requestJSON[T](t, http.MethodPost, url, payload, headers, http.StatusCreated, http.StatusAccepted, http.StatusOK)
}

func putJSONWithHeaders[T any](t *testing.T, url string, payload any, headers map[string]string) T {
	t.Helper()
	return requestJSON[T](t, http.MethodPut, url, payload, headers, http.StatusOK)
}

func putJSONExpectStatus(t *testing.T, url string, payload any, headers map[string]string, expectedStatus int) {
	t.Helper()
	requestStatus(t, http.MethodPut, url, payload, headers, expectedStatus)
}

func getJSON[T any](t *testing.T, url string) T {
	t.Helper()
	return getJSONWithHeaders[T](t, url, nil)
}

func getJSONWithHeaders[T any](t *testing.T, url string, headers map[string]string) T {
	t.Helper()
	return requestJSON[T](t, http.MethodGet, url, nil, headers, http.StatusOK)
}

func requestJSON[T any](t *testing.T, method, url string, payload any, headers map[string]string, expectedStatuses ...int) T {
	t.Helper()
	response := doRequest(t, method, url, payload, headers, expectedStatuses...)
	defer response.Body.Close()

	var result T
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func requestStatus(t *testing.T, method, url string, payload any, headers map[string]string, expectedStatus int) {
	t.Helper()
	response := doRequest(t, method, url, payload, headers, expectedStatus)
	defer response.Body.Close()
}

func doRequest(t *testing.T, method, url string, payload any, headers map[string]string, expectedStatuses ...int) *http.Response {
	t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s request: %v", strings.ToLower(method), err)
	}

	for _, status := range expectedStatuses {
		if response.StatusCode == status {
			return response
		}
	}

	bodyText, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	t.Fatalf("unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyText)))
	return nil
}

func setupIntegrationPrereqs(t *testing.T) (context.Context, string, *gorm.DB) {
	t.Helper()

	testDatabaseDSN := strings.TrimSpace(os.Getenv("OCTOMANGER_TEST_DATABASE_DSN"))
	if testDatabaseDSN == "" {
		t.Skip("OCTOMANGER_TEST_DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	rootDir := repoRoot(t)
	db := openSchemaDB(t, ctx, testDatabaseDSN)
	if err := database.AutoMigrate(ctx, db); err != nil {
		t.Fatalf("auto migrate database: %v", err)
	}
	return ctx, rootDir, db
}

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}
}

func startFakeServer(t *testing.T, ctx context.Context, rootDir string) string {
	t.Helper()

	requirePython(t)
	port := reservePort(t)
	scriptPath := filepath.Join(rootDir, "plugins/modules/octo_demo/fake_server.py")
	cmd := exec.CommandContext(ctx, "python3", scriptPath, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fake server: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForHTTPStatus(t, ctx, baseURL+"/tasks", http.StatusUnauthorized)
	return baseURL
}

func startTestAPI(t *testing.T, ctx context.Context, rootDir string, db *gorm.DB, adminKey string) *testAPI {
	t.Helper()

	logger := logging.New(platformconfig.LoggingConfig{
		Level:  "error",
		Format: "text",
	})
	plugins := pluginapp.New(
		fsrepo.New(filepath.Join(rootDir, "plugins/modules")),
		"python3",
		filepath.Join(rootDir, "plugins/sdk/python"),
	)
	accountTypes := accounttypeapp.New(accounttypepostgres.New(db))
	accounts := accountapp.New(accountpostgres.New(db))
	email := emailapp.New(emailpostgres.New(db))
	jobs := jobapp.New(logger, jobpostgres.New(db), plugins, "test-worker")
	triggers := triggerapp.New(triggerpostgres.New(db), jobs)
	agents := agentapp.New(logger, agentpostgres.New(db), plugins, nil, "test-worker", 50*time.Millisecond, 50*time.Millisecond)
	system := systemapp.New(db, plugins)

	addr := fmt.Sprintf("127.0.0.1:%d", reservePort(t))
	h := server.New(
		server.WithHostPorts(addr),
		server.WithExitWaitTime(time.Second),
	)

	root := h.Group("/")
	v2 := h.Group("/api/v2")
	systemtransport.NewHandler(adminKey, system).Register(root, v2)
	plugintransport.NewHandler(adminKey, plugins, accountTypes, system).Register(v2)
	jobtransport.NewHandler(adminKey, jobs).Register(v2)
	agenttransport.NewHandler(adminKey, agents).Register(v2)
	accounttypestransport.NewHandler(adminKey, accountTypes).Register(v2)
	accountstransport.NewHandler(adminKey, accounts, plugins).Register(v2)
	emailtransport.NewHandler(adminKey, email).Register(v2)
	triggertransport.NewHandler(adminKey, triggers).Register(v2, root)

	go h.Spin()
	t.Cleanup(func() {
		h.Shutdown(context.Background()) //nolint:errcheck
	})

	baseURL := "http://" + addr
	waitForHTTPStatus(t, ctx, baseURL+"/healthz", http.StatusOK)
	return &testAPI{
		baseURL: baseURL,
		jobs:    jobs,
		db:      db,
	}
}

func waitForHTTPStatus(t *testing.T, ctx context.Context, url string, expectedStatus int) {
	t.Helper()

	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("new wait request: %v", err)
		}

		response, err := http.DefaultClient.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == expectedStatus {
				return
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for %s status %d timed out", url, expectedStatus)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func reservePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func adminHeaders(adminKey string) map[string]string {
	return map[string]string{
		"X-Admin-Key": adminKey,
	}
}

func openSchemaDB(t *testing.T, ctx context.Context, baseDSN string) *gorm.DB {
	t.Helper()

	adminPool, err := pgxpool.New(ctx, baseDSN)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schemaName := fmt.Sprintf("octomanger_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName))
	})

	db, err := database.Open(platformconfig.DatabaseConfig{
		DSN:            withSearchPath(baseDSN, schemaName),
		MaxConnections: 4,
		QueryTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("open schema db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func withSearchPath(baseDSN, schemaName string) string {
	if strings.Contains(baseDSN, "://") {
		parsed, err := url.Parse(baseDSN)
		if err == nil {
			query := parsed.Query()
			query.Set("search_path", schemaName)
			parsed.RawQuery = query.Encode()
			return parsed.String()
		}
	}
	return strings.TrimSpace(baseDSN) + " search_path=" + schemaName
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestResolveStaticFilePrefersBrotliAssets(t *testing.T) {
	distDir := t.TempDir()
	assetPath := filepath.Join(distDir, "assets", "app.js")
	writeTestFile(t, assetPath, []byte("console.log('ok');"))
	writeTestFile(t, assetPath+".gz", []byte("gzip"))
	writeTestFile(t, assetPath+".br", []byte("brotli"))

	file, ok := resolveStaticFile(distDir, http.MethodGet, "/assets/app.js", "*/*", "br, gzip")
	if !ok {
		t.Fatal("expected static asset to resolve")
	}
	if file.diskPath != assetPath+".br" {
		t.Fatalf("unexpected disk path %q", file.diskPath)
	}
	if file.contentEncoding != "br" {
		t.Fatalf("unexpected content encoding %q", file.contentEncoding)
	}
	if file.cacheControl != cacheControlImmutableAssets {
		t.Fatalf("unexpected cache control %q", file.cacheControl)
	}
	if !strings.Contains(file.contentType, "javascript") {
		t.Fatalf("unexpected content type %q", file.contentType)
	}
}

func TestResolveStaticFileUsesSPAIndexForHTMLNavigations(t *testing.T) {
	distDir := t.TempDir()
	indexPath := filepath.Join(distDir, "index.html")
	writeTestFile(t, indexPath, []byte("<!doctype html>"))

	file, ok := resolveStaticFile(distDir, http.MethodGet, "/jobs/42", "text/html,application/xhtml+xml", "gzip")
	if !ok {
		t.Fatal("expected SPA fallback to resolve")
	}
	if file.diskPath != indexPath {
		t.Fatalf("unexpected disk path %q", file.diskPath)
	}
	if file.cacheControl != cacheControlNoCache {
		t.Fatalf("unexpected cache control %q", file.cacheControl)
	}
}

func TestResolveStaticFileDoesNotFallbackForMissingAssetsOrAPIPaths(t *testing.T) {
	distDir := t.TempDir()
	writeTestFile(t, filepath.Join(distDir, "index.html"), []byte("<!doctype html>"))

	if _, ok := resolveStaticFile(distDir, http.MethodGet, "/assets/missing.js", "text/html", "gzip"); ok {
		t.Fatal("expected missing asset request to return 404")
	}
	if _, ok := resolveStaticFile(distDir, http.MethodGet, "/api/v2/unknown", "text/html", "gzip"); ok {
		t.Fatal("expected api path to return 404")
	}
	if _, ok := resolveStaticFile(distDir, http.MethodPost, "/jobs/42", "text/html", "gzip"); ok {
		t.Fatal("expected non-GET html request to return 404")
	}
}

func TestAcceptsEncodingHonorsZeroQuality(t *testing.T) {
	if acceptsEncoding("br;q=0, gzip;q=1", "br") {
		t.Fatal("expected brotli with q=0 to be rejected")
	}
	if !acceptsEncoding("br;q=0, gzip;q=1", "gzip") {
		t.Fatal("expected gzip to remain accepted")
	}
}

func writeTestFile(t *testing.T, filename string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filename, err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}
