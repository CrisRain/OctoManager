package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB) error {
	if err := renameLegacySystemConfigColumn(ctx, db); err != nil {
		return err
	}

	if err := db.WithContext(ctx).AutoMigrate(
		&AccountTypeModel{},
		&AccountModel{},
		&EmailAccountModel{},
		&JobDefinitionModel{},
		&ScheduleModel{},
		&JobExecutionModel{},
		&JobLogModel{},
		&TriggerModel{},
		&AgentModel{},
		&AgentLogModel{},
		&SystemConfigModel{},
	); err != nil {
		return fmt.Errorf("auto migrate database schema: %w", err)
	}

	if err := ensureIndexes(ctx, db); err != nil {
		return err
	}
	return nil
}

func renameLegacySystemConfigColumn(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Exec(`
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'system_configs' AND column_name = 'value'
			) AND NOT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = 'system_configs' AND column_name = 'value_json'
			) THEN
				ALTER TABLE system_configs RENAME COLUMN value TO value_json;
			END IF;
		END $$;
	`).Error
}

func ensureIndexes(ctx context.Context, db *gorm.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_schedules_next_run_at ON schedules (next_run_at) WHERE enabled = TRUE`,
		`CREATE INDEX IF NOT EXISTS idx_job_executions_def_id_created ON job_executions (job_definition_id, created_at DESC)`,
	}
	for _, query := range indexes {
		if err := db.WithContext(ctx).Exec(query).Error; err != nil {
			return fmt.Errorf("ensure database indexes: %w", err)
		}
	}
	return nil
}
