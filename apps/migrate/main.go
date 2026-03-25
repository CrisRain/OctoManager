package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
)

func main() {
	mode := flag.String("mode", "migrate", "migrate | import-legacy")
	legacyDSN := flag.String("legacy-dsn", strings.TrimSpace(os.Getenv("LEGACY_DATABASE_URL")), "legacy database DSN")
	flag.Parse()

	timeout := 30 * time.Second
	if *mode == "import-legacy" {
		timeout = 10 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg := config.MustLoad()
	targetDB, err := database.Open(cfg.Database)
	if err != nil {
		panic(err)
	}
	defer func() {
		if sqlDB, err := targetDB.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	if err := database.AutoMigrate(ctx, targetDB); err != nil {
		panic(err)
	}

	switch *mode {
	case "migrate":
		return
	case "import-legacy":
		if *legacyDSN == "" {
			panic("legacy DSN is required for import-legacy mode")
		}

		legacyDB, err := gorm.Open(postgres.Open(*legacyDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			panic(fmt.Sprintf("open legacy database: %v", err))
		}
		defer func() {
			if sqlDB, err := legacyDB.DB(); err == nil {
				sqlDB.Close()
			}
		}()

		if err := importLegacy(ctx, legacyDB, targetDB); err != nil {
			panic(err)
		}
	default:
		panic(fmt.Sprintf("unsupported mode %q", *mode))
	}
}

func importLegacy(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB) error {
	typeMap, err := importLegacyAccountTypes(ctx, legacyDB, targetDB)
	if err != nil {
		return err
	}
	if err := importLegacyAccounts(ctx, legacyDB, targetDB, typeMap); err != nil {
		return err
	}
	if err := importLegacyEmailAccounts(ctx, legacyDB, targetDB); err != nil {
		return err
	}
	jobMap, err := importLegacyJobs(ctx, legacyDB, targetDB)
	if err != nil {
		return err
	}
	if err := importLegacyJobRuns(ctx, legacyDB, targetDB, jobMap); err != nil {
		return err
	}
	if err := importLegacyTriggers(ctx, legacyDB, targetDB); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "legacy import complete")
	return nil
}

func importLegacyAccountTypes(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB) (map[string]int64, error) {
	rows, err := legacyDB.WithContext(ctx).Raw(`SELECT key, name, category, schema, capabilities FROM account_types ORDER BY id ASC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query legacy account types: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, name, category string
		var schemaJSON, capabilityJSON []byte
		if err := rows.Scan(&key, &name, &category, &schemaJSON, &capabilityJSON); err != nil {
			return nil, fmt.Errorf("scan legacy account type: %w", err)
		}
		if result := targetDB.WithContext(ctx).Exec(`
			INSERT INTO account_types (key, name, category, schema_json, capabilities_json)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (key) DO UPDATE
			SET name = EXCLUDED.name, category = EXCLUDED.category,
			    schema_json = EXCLUDED.schema_json, capabilities_json = EXCLUDED.capabilities_json, updated_at = NOW()`,
			key, name, defaultString(category, "generic"),
			defaultJSON(schemaJSON, `{}`), defaultJSON(capabilityJSON, `{}`),
		); result.Error != nil {
			return nil, fmt.Errorf("upsert account type %s: %w", key, result.Error)
		}
	}

	typeMap := map[string]int64{}
	typeRows, err := targetDB.WithContext(ctx).Raw(`SELECT id, key FROM account_types`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query imported account types: %w", err)
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var id int64
		var key string
		if err := typeRows.Scan(&id, &key); err != nil {
			return nil, err
		}
		typeMap[key] = id
	}
	return typeMap, nil
}

func importLegacyAccounts(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB, typeMap map[string]int64) error {
	rows, err := legacyDB.WithContext(ctx).Raw(`SELECT type_key, identifier, status, tags, spec FROM accounts ORDER BY id ASC`).Rows()
	if err != nil {
		return fmt.Errorf("query legacy accounts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var typeKey, identifier string
		var status int16
		var tags []string
		var specJSON []byte
		if err := rows.Scan(&typeKey, &identifier, &status, &tags, &specJSON); err != nil {
			return fmt.Errorf("scan legacy account: %w", err)
		}
		var accountTypeID any
		if v, ok := typeMap[typeKey]; ok {
			accountTypeID = v
		}
		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			return err
		}
		if result := targetDB.WithContext(ctx).Exec(`
			INSERT INTO accounts (account_type_id, identifier, status, tags_json, spec_json)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (account_type_id, identifier) DO UPDATE
			SET status = EXCLUDED.status, tags_json = EXCLUDED.tags_json,
			    spec_json = EXCLUDED.spec_json, updated_at = NOW()`,
			accountTypeID, identifier, legacyStatus(status), tagsJSON, defaultJSON(specJSON, `{}`),
		); result.Error != nil {
			return fmt.Errorf("upsert account %s: %w", identifier, result.Error)
		}
	}
	return nil
}

func importLegacyEmailAccounts(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB) error {
	rows, err := legacyDB.WithContext(ctx).Raw(`SELECT address, provider, status, graph_config FROM email_accounts ORDER BY id ASC`).Rows()
	if err != nil {
		return fmt.Errorf("query legacy email accounts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var address, provider string
		var status int16
		var configJSON []byte
		if err := rows.Scan(&address, &provider, &status, &configJSON); err != nil {
			return fmt.Errorf("scan legacy email account: %w", err)
		}
		if result := targetDB.WithContext(ctx).Exec(`
			INSERT INTO email_accounts (address, provider, status, config_json)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (address) DO UPDATE
			SET provider = EXCLUDED.provider, status = EXCLUDED.status,
			    config_json = EXCLUDED.config_json, updated_at = NOW()`,
			address, defaultString(provider, "outlook"), legacyStatus(status), defaultJSON(configJSON, `{}`),
		); result.Error != nil {
			return fmt.Errorf("upsert email account %s: %w", address, result.Error)
		}
	}
	return nil
}

func importLegacyJobs(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB) (map[uint64]int64, error) {
	rows, err := legacyDB.WithContext(ctx).Raw(`SELECT id, type_key, action_key, params FROM jobs ORDER BY id ASC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query legacy jobs: %w", err)
	}
	defer rows.Close()

	imported := map[uint64]int64{}
	for rows.Next() {
		var id uint64
		var typeKey, actionKey string
		var paramsRaw []byte
		if err := rows.Scan(&id, &typeKey, &actionKey, &paramsRaw); err != nil {
			return nil, fmt.Errorf("scan legacy job: %w", err)
		}
		params := decodeJSONMap(paramsRaw)
		scheduleExpr := popString(params, "_schedule")
		timezone := popString(params, "_timezone")

		var definitionID int64
		row := targetDB.WithContext(ctx).Raw(`
			INSERT INTO job_definitions (key, name, plugin_key, action, input_json, enabled)
			VALUES ($1, $2, $3, $4, $5, TRUE)
			ON CONFLICT (key) DO UPDATE
			SET name = EXCLUDED.name, plugin_key = EXCLUDED.plugin_key,
			    action = EXCLUDED.action, input_json = EXCLUDED.input_json, updated_at = NOW()
			RETURNING id`,
			fmt.Sprintf("legacy-job-%d", id),
			fmt.Sprintf("%s:%s #%d", typeKey, actionKey, id),
			typeKey, actionKey, mustJSON(params),
		).Row()
		if err := row.Scan(&definitionID); err != nil {
			return nil, fmt.Errorf("upsert legacy job definition %d: %w", id, err)
		}
		imported[id] = definitionID

		if scheduleExpr != "" {
			nextRunAt := time.Now().UTC()
			if result := targetDB.WithContext(ctx).Exec(`
				INSERT INTO schedules (job_definition_id, cron_expression, timezone, next_run_at, enabled)
				VALUES ($1, $2, $3, $4, TRUE)
				ON CONFLICT (job_definition_id) DO UPDATE
				SET cron_expression = EXCLUDED.cron_expression, timezone = EXCLUDED.timezone,
				    next_run_at = EXCLUDED.next_run_at, enabled = TRUE, updated_at = NOW()`,
				definitionID, scheduleExpr, defaultString(timezone, "UTC"), nextRunAt,
			); result.Error != nil {
				return nil, fmt.Errorf("upsert legacy schedule %d: %w", id, result.Error)
			}
		}
	}
	return imported, nil
}

func importLegacyTriggers(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB) error {
	rows, err := legacyDB.WithContext(ctx).Raw(`
		SELECT slug, name, type_key, action_key, execution_mode, default_params, token_hash, token_prefix, enabled
		FROM trigger_endpoints ORDER BY id ASC`).Rows()
	if err != nil {
		return fmt.Errorf("query legacy triggers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var slug, name, typeKey, actionKey, mode, tokenHash, tokenPrefix string
		var defaultParams []byte
		var enabled bool
		if err := rows.Scan(&slug, &name, &typeKey, &actionKey, &mode, &defaultParams, &tokenHash, &tokenPrefix, &enabled); err != nil {
			return fmt.Errorf("scan legacy trigger: %w", err)
		}
		var definitionID int64
		row := targetDB.WithContext(ctx).Raw(`
			INSERT INTO job_definitions (key, name, plugin_key, action, input_json, enabled)
			VALUES ($1, $2, $3, $4, $5, TRUE)
			ON CONFLICT (key) DO UPDATE
			SET name = EXCLUDED.name, plugin_key = EXCLUDED.plugin_key,
			    action = EXCLUDED.action, input_json = EXCLUDED.input_json, updated_at = NOW()
			RETURNING id`,
			"legacy-trigger-"+slug, name, typeKey, actionKey, defaultJSON(defaultParams, `{}`),
		).Row()
		if err := row.Scan(&definitionID); err != nil {
			return fmt.Errorf("upsert trigger definition %s: %w", slug, err)
		}
		if result := targetDB.WithContext(ctx).Exec(`
			INSERT INTO triggers (key, name, job_definition_id, mode, default_input_json, token_hash, token_prefix, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (key) DO UPDATE
			SET name = EXCLUDED.name, job_definition_id = EXCLUDED.job_definition_id,
			    mode = EXCLUDED.mode, default_input_json = EXCLUDED.default_input_json,
			    token_hash = EXCLUDED.token_hash, token_prefix = EXCLUDED.token_prefix,
			    enabled = EXCLUDED.enabled, updated_at = NOW()`,
			slug, name, definitionID, defaultString(mode, "async"),
			defaultJSON(defaultParams, `{}`), tokenHash, tokenPrefix, enabled,
		); result.Error != nil {
			return fmt.Errorf("upsert trigger %s: %w", slug, result.Error)
		}
	}
	return nil
}

func importLegacyJobRuns(ctx context.Context, legacyDB *gorm.DB, targetDB *gorm.DB, jobMap map[uint64]int64) error {
	rows, err := legacyDB.WithContext(ctx).Raw(`
		SELECT id, job_id, account_id, worker_id, attempt, result, logs, error_code, error_message, started_at, ended_at
		FROM job_runs ORDER BY id ASC`).Rows()
	if err != nil {
		return fmt.Errorf("query legacy job runs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var runID, jobID uint64
		var accountID *uint64
		var workerID, errorCode, errorMessage string
		var attempt int
		var resultJSON, logsJSON []byte
		var startedAt time.Time
		var endedAt *time.Time
		if err := rows.Scan(&runID, &jobID, &accountID, &workerID, &attempt, &resultJSON, &logsJSON, &errorCode, &errorMessage, &startedAt, &endedAt); err != nil {
			return fmt.Errorf("scan legacy job run: %w", err)
		}
		definitionID, ok := jobMap[jobID]
		if !ok {
			continue
		}
		status := importedExecutionStatus(errorCode, errorMessage, endedAt)
		resultPayload := decodeJSONMap(resultJSON)
		if accountID != nil {
			resultPayload["legacy_account_id"] = *accountID
		}
		resultPayload["legacy_attempt"] = attempt
		resultPayload["legacy_job_run_id"] = runID

		executionID, err := upsertImportedExecution(ctx, targetDB, definitionID, runID, workerID, status, resultPayload, errorMessage, startedAt, endedAt)
		if err != nil {
			return err
		}
		logCount := appendImportedLogsSummary(ctx, targetDB, executionID, runID, accountID, logsJSON, errorCode, errorMessage)
		targetDB.WithContext(ctx).Exec(`
			UPDATE job_executions SET summary = $2, updated_at = NOW() WHERE id = $1`,
			executionID,
			fmt.Sprintf("imported legacy run #%d with %d log lines", runID, logCount),
		)
	}
	return nil
}

func upsertImportedExecution(
	ctx context.Context,
	targetDB *gorm.DB,
	definitionID int64,
	runID uint64,
	workerID, status string,
	resultPayload map[string]any,
	errorMessage string,
	startedAt time.Time,
	endedAt *time.Time,
) (int64, error) {
	var existingID int64
	row := targetDB.WithContext(ctx).Raw(`
		SELECT id FROM job_executions
		WHERE source = 'legacy-import'
			AND job_definition_id = $1
			AND result_json ->> 'legacy_job_run_id' = $2`,
		definitionID, fmt.Sprintf("%d", runID),
	).Row()
	if err := row.Scan(&existingID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("check imported execution %d: %w", runID, err)
		}
	} else {
		return existingID, nil
	}

	var executionID int64
	insRow := targetDB.WithContext(ctx).Raw(`
		INSERT INTO job_executions (
			job_definition_id, input_json, status, requested_by, source,
			worker_id, summary, result_json, error_message,
			started_at, finished_at, created_at, updated_at
		)
		SELECT id, input_json, $2, $3, $4, $5, $6, $7, $8, $9, $10, $9, COALESCE($10, $9)
		FROM job_definitions WHERE id = $1
		RETURNING id`,
		definitionID, status, "import:legacy", "legacy-import",
		workerID, fmt.Sprintf("imported legacy run #%d", runID),
		mustJSON(resultPayload), errorMessage, startedAt, endedAt,
	).Row()
	if err := insRow.Scan(&executionID); err != nil {
		return 0, fmt.Errorf("insert imported execution %d: %w", runID, err)
	}
	return executionID, nil
}

func appendImportedLogsSummary(ctx context.Context, targetDB *gorm.DB, executionID int64, runID uint64, accountID *uint64, logsJSON []byte, errorCode, errorMessage string) int {
	logLines := decodeJSONStringArray(logsJSON)
	payload := map[string]any{
		"legacy_job_run_id": runID,
		"log_count":         len(logLines),
	}
	if accountID != nil {
		payload["legacy_account_id"] = *accountID
	}
	if errorCode != "" {
		payload["legacy_error_code"] = errorCode
	}
	if errorMessage != "" {
		payload["legacy_error_message"] = errorMessage
	}
	targetDB.WithContext(ctx).Exec(`
		INSERT INTO job_logs (job_execution_id, stream, event_type, message, payload_json)
		VALUES ($1, $2, $3, $4, $5)`,
		executionID, "legacy", "legacy_summary",
		fmt.Sprintf("imported legacy job run #%d", runID), mustJSON(payload),
	)
	return len(logLines)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultJSON(value []byte, fallback string) []byte {
	if len(value) == 0 {
		return []byte(fallback)
	}
	return value
}

func decodeJSONMap(value []byte) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := map[string]any{}
	_ = json.Unmarshal(value, &result)
	return result
}

func decodeJSONStringArray(value []byte) []string {
	if len(value) == 0 {
		return []string{}
	}
	var result []string
	_ = json.Unmarshal(value, &result)
	return result
}

func popString(values map[string]any, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}
	delete(values, key)
	if text, ok := raw.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func mustJSON(value map[string]any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}

func legacyStatus(value int16) string {
	switch value {
	case 1:
		return "active"
	case 2:
		return "disabled"
	default:
		return "inactive"
	}
}

func importedExecutionStatus(errorCode, errorMessage string, endedAt *time.Time) string {
	if strings.TrimSpace(errorCode) != "" || strings.TrimSpace(errorMessage) != "" {
		return "failed"
	}
	if endedAt == nil {
		return "running"
	}
	return "succeeded"
}
