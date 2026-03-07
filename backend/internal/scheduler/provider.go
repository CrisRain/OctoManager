package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"octomanger/backend/internal/model"
	"octomanger/backend/internal/task"
)

// DBProvider implements asynq.PeriodicTaskConfigProvider.
// It queries jobs whose params JSON contains a "_schedule" cron expression,
// and registers them as recurring asynq tasks without any changes to existing code.
type DBProvider struct {
	db *gorm.DB
}

func NewDBProvider(db *gorm.DB) *DBProvider {
	return &DBProvider{db: db}
}

// GetConfigs is called by asynq.PeriodicTaskManager on each sync interval.
// Jobs with params["_schedule"] = "<cron expr>" are returned as periodic configs.
func (p *DBProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	var jobs []model.Job
	err := p.db.
		WithContext(context.Background()).
		Where("params->>'_schedule' IS NOT NULL AND params->>'_schedule' != ''").
		Find(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("scheduler: query scheduled jobs: %w", err)
	}

	configs := make([]*asynq.PeriodicTaskConfig, 0, len(jobs))
	for _, job := range jobs {
		var params map[string]any
		if err := json.Unmarshal(job.Params, &params); err != nil {
			continue
		}
		cronspec, _ := params["_schedule"].(string)
		if cronspec == "" {
			continue
		}

		payload, err := json.Marshal(task.DispatchJobPayload{JobID: job.ID})
		if err != nil {
			continue
		}

		configs = append(configs, &asynq.PeriodicTaskConfig{
			Cronspec: cronspec,
			Task:     asynq.NewTask(task.TypeDispatchJob, payload),
			Opts: []asynq.Option{
				asynq.Queue("default"),
				asynq.MaxRetry(3),
				// Unique task ID per job prevents duplicate enqueues if the
				// previous run hasn't been picked up yet.
				asynq.TaskID(fmt.Sprintf("scheduled-job-%d", job.ID)),
			},
		})
	}
	return configs, nil
}
