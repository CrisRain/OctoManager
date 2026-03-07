package database

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"octomanger/backend/internal/model"
)

type MigrateOptions struct {
	DropLegacy bool
}

type MigrationReport struct {
	DroppedTables  []string
	DroppedColumns []string
}

func Migrate(db *gorm.DB, opts MigrateOptions) (MigrationReport, error) {
	var report MigrationReport
	if db == nil {
		return report, errors.New("db is nil")
	}

	hasUUID, hasTenant, err := detectLegacySchema(db)
	if err != nil {
		return report, err
	}

	if hasUUID {
		if !opts.DropLegacy {
			return report, errors.New("legacy uuid schema detected; set database.reset=true to rebuild")
		}
		dropped, err := dropAllTables(db)
		if err != nil {
			return report, err
		}
		report.DroppedTables = dropped
	} else if hasTenant {
		columns, err := dropLegacyColumns(db)
		if err != nil {
			return report, err
		}
		report.DroppedColumns = columns
	}

	graphColumns, err := migrateEmailAccountGraphConfigColumn(db)
	if err != nil {
		return report, err
	}
	report.DroppedColumns = append(report.DroppedColumns, graphColumns...)

	smtpColumns, err := dropEmailAccountSMTPConfigColumn(db)
	if err != nil {
		return report, err
	}
	report.DroppedColumns = append(report.DroppedColumns, smtpColumns...)

	if err := normalizeEmailAccountGraphConfig(db); err != nil {
		return report, err
	}

	if err := db.AutoMigrate(
		&model.AccountType{},
		&model.Account{},
		&model.AccountSession{},
		&model.EmailAccount{},
		&model.Job{},
		&model.JobRun{},
		&model.ApiKey{},
		&model.TriggerEndpoint{},
		&model.SystemConfig{},
	); err != nil {
		return report, err
	}

	if err := normalizeTriggerExecutionMode(db); err != nil {
		return report, err
	}

	if err := cleanupSeededDefaults(db); err != nil {
		return report, err
	}

	if err := seedSystemConfigs(db); err != nil {
		return report, err
	}

	return report, nil
}

func migrateEmailAccountGraphConfigColumn(db *gorm.DB) ([]string, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if !db.Migrator().HasTable(&model.EmailAccount{}) {
		return nil, nil
	}

	hasOld := db.Migrator().HasColumn(&model.EmailAccount{}, "imap_config")
	hasNew := db.Migrator().HasColumn(&model.EmailAccount{}, "graph_config")

	switch {
	case hasOld && !hasNew:
		if err := db.Migrator().RenameColumn(&model.EmailAccount{}, "imap_config", "graph_config"); err != nil {
			return nil, err
		}
		return []string{"email_accounts.imap_config (renamed)"}, nil
	case hasOld && hasNew:
		if err := db.Exec(`
			UPDATE email_accounts
			SET graph_config = imap_config
			WHERE (graph_config IS NULL OR graph_config = '{}'::jsonb)
			  AND imap_config IS NOT NULL
		`).Error; err != nil {
			return nil, err
		}
		if err := db.Migrator().DropColumn(&model.EmailAccount{}, "imap_config"); err != nil {
			return nil, err
		}
		return []string{"email_accounts.imap_config"}, nil
	default:
		return nil, nil
	}
}

func normalizeEmailAccountGraphConfig(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	if !db.Migrator().HasTable(&model.EmailAccount{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&model.EmailAccount{}, "graph_config") {
		return nil
	}

	return db.Exec(`
		UPDATE email_accounts
		SET graph_config =
			(graph_config - 'token' - 'expires_at' - 'folder' - 'user')
			|| CASE
				WHEN graph_config ? 'token' AND NOT graph_config ? 'access_token'
				THEN jsonb_build_object('access_token', graph_config->'token')
				ELSE '{}'::jsonb
			END
			|| CASE
				WHEN graph_config ? 'expires_at' AND NOT graph_config ? 'token_expires_at'
				THEN jsonb_build_object('token_expires_at', graph_config->'expires_at')
				ELSE '{}'::jsonb
			END
			|| CASE
				WHEN graph_config ? 'folder' AND NOT graph_config ? 'mailbox'
				THEN jsonb_build_object('mailbox', graph_config->'folder')
				ELSE '{}'::jsonb
			END
			|| CASE
				WHEN graph_config ? 'user' AND NOT graph_config ? 'username'
				THEN jsonb_build_object('username', graph_config->'user')
				ELSE '{}'::jsonb
			END
		WHERE graph_config IS NOT NULL
	`).Error
}

func dropEmailAccountSMTPConfigColumn(db *gorm.DB) ([]string, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if !db.Migrator().HasTable(&model.EmailAccount{}) {
		return nil, nil
	}
	if !db.Migrator().HasColumn(&model.EmailAccount{}, "smtp_config") {
		return nil, nil
	}
	if err := db.Migrator().DropColumn(&model.EmailAccount{}, "smtp_config"); err != nil {
		return nil, err
	}
	return []string{"email_accounts.smtp_config"}, nil
}

func detectLegacySchema(db *gorm.DB) (bool, bool, error) {
	hasUUID := false
	hasTenant := false

	models := []any{
		&model.AccountType{},
		&model.Account{},
		&model.AccountSession{},
		&model.EmailAccount{},
		&model.Job{},
		&model.JobRun{},
		&model.ApiKey{},
		&model.TriggerEndpoint{},
	}

	for _, m := range models {
		if !db.Migrator().HasTable(m) {
			continue
		}
		if db.Migrator().HasColumn(m, "tenant_id") {
			hasTenant = true
		}
		uuidPrimaryKey, err := hasUUIDPrimaryKey(db, m)
		if err != nil {
			return false, false, err
		}
		if uuidPrimaryKey {
			hasUUID = true
		}
	}

	return hasUUID, hasTenant, nil
}

func hasUUIDPrimaryKey(db *gorm.DB, m any) (bool, error) {
	columnTypes, err := db.Migrator().ColumnTypes(m)
	if err != nil {
		return false, err
	}
	for _, col := range columnTypes {
		name := strings.ToLower(col.Name())
		if name != "id" {
			continue
		}
		typeName := strings.ToLower(col.DatabaseTypeName())
		if strings.Contains(typeName, "uuid") {
			return true, nil
		}
	}
	return false, nil
}

func dropLegacyColumns(db *gorm.DB) ([]string, error) {
	var dropped []string

	targets := []struct {
		model  any
		column string
	}{
		{&model.AccountType{}, "tenant_id"},
		{&model.Account{}, "tenant_id"},
		{&model.AccountSession{}, "tenant_id"},
		{&model.EmailAccount{}, "tenant_id"},
		{&model.Job{}, "tenant_id"},
		{&model.JobRun{}, "tenant_id"},
		{&model.ApiKey{}, "tenant_id"},
		{&model.TriggerEndpoint{}, "tenant_id"},
		{&model.SystemConfig{}, "tenant_id"},
	}

	for _, target := range targets {
		if !db.Migrator().HasTable(target.model) {
			continue
		}
		if db.Migrator().HasColumn(target.model, target.column) {
			if err := db.Migrator().DropColumn(target.model, target.column); err != nil {
				return dropped, err
			}
			dropped = append(dropped, target.column)
		}
	}

	return dropped, nil
}

func dropAllTables(db *gorm.DB) ([]string, error) {
	order := []struct {
		model any
		name  string
	}{
		{&model.JobRun{}, "job_runs"},
		{&model.AccountSession{}, "account_sessions"},
		{&model.Job{}, "jobs"},
		{&model.TriggerEndpoint{}, "trigger_endpoints"},
		{&model.ApiKey{}, "api_keys"},
		{&model.EmailAccount{}, "email_accounts"},
		{&model.Account{}, "accounts"},
		{&model.AccountType{}, "account_types"},
		{&model.SystemConfig{}, "system_configs"},
	}

	dropped := make([]string, 0, len(order))
	for _, item := range order {
		if db.Migrator().HasTable(item.model) {
			if err := db.Migrator().DropTable(item.model); err != nil {
				return dropped, err
			}
			dropped = append(dropped, item.name)
		}
	}

	return dropped, nil
}

func cleanupSeededDefaults(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	if !db.Migrator().HasTable(&model.AccountType{}) {
		return nil
	}
	return db.Exec(`
		DELETE FROM account_types
		WHERE key = 'email'
		  AND name = 'Email Account'
		  AND category = 'email'
		  AND version = 1
		  AND schema = '{}'::jsonb
		  AND capabilities = '{"actions":[{"key":"VERIFY"},{"key":"SEND"},{"key":"FETCH"},{"key":"HEALTH_CHECK"}]}'::jsonb
		  AND (script_config IS NULL OR script_config = 'null'::jsonb)
	`).Error
}

func seedSystemConfigs(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	if !db.Migrator().HasTable(&model.SystemConfig{}) {
		return nil
	}
	defaults := []struct {
		key         string
		value       string
		description string
		critical    bool
	}{
		{"app.name", `"OctoManger"`, "Display name of the application", false},
		{"job.default_timeout_minutes", `30`, "Default job execution timeout in minutes", false},
		{"job.max_concurrency", `10`, "Maximum concurrent jobs per worker", false},
	}
	for _, d := range defaults {
		if err := db.Exec(`
			INSERT INTO system_configs (key, value, description, is_critical, created_at, updated_at)
			VALUES (?, ?::jsonb, ?, ?, NOW(), NOW())
			ON CONFLICT (key) DO NOTHING
		`, d.key, d.value, d.description, d.critical).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeTriggerExecutionMode(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	if !db.Migrator().HasTable(&model.TriggerEndpoint{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&model.TriggerEndpoint{}, "execution_mode") {
		return nil
	}
	return db.Exec(`
		UPDATE trigger_endpoints
		SET execution_mode = 'async'
		WHERE execution_mode IS NULL OR btrim(execution_mode) = ''
	`).Error
}
