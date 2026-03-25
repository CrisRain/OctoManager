package runtime

import (
	"context"

	"go.uber.org/zap"
)

func enforceLogRetentionOnStartup(ctx context.Context, app *App) {
	queries := []struct {
		name  string
		query string
	}{
		{
			name: "job_logs",
			query: `
				DELETE FROM job_logs
				WHERE id IN (
					SELECT id
					FROM (
						SELECT id,
						       ROW_NUMBER() OVER (PARTITION BY job_execution_id ORDER BY id DESC) AS rn
						FROM job_logs
					) ranked
					WHERE rn > 200
				)`,
		},
		{
			name: "agent_logs",
			query: `
				DELETE FROM agent_logs
				WHERE id IN (
					SELECT id
					FROM (
						SELECT id,
						       ROW_NUMBER() OVER (PARTITION BY agent_id ORDER BY id DESC) AS rn
						FROM agent_logs
					) ranked
					WHERE rn > 200
				)`,
		},
	}

	for _, item := range queries {
		result := app.DB.WithContext(ctx).Exec(item.query)
		if result.Error != nil {
			app.Logger.Warn("enforce log retention failed", zap.String("table", item.name), zap.Error(result.Error))
			continue
		}
		if result.RowsAffected > 0 {
			app.Logger.Info("enforced log retention", zap.String("table", item.name), zap.Int64("deleted_rows", result.RowsAffected))
		}
	}
}
