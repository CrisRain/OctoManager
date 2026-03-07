package repository

import (
	"context"

	"gorm.io/gorm"
	"octomanger/backend/internal/model"
)

type JobRunRepository interface {
	Create(ctx context.Context, item *model.JobRun) error
	GetByID(ctx context.Context, id uint64) (*model.JobRun, error)
	GetLatestByJobID(ctx context.Context, jobID uint64) (*model.JobRun, error)
	ListByJobID(ctx context.Context, jobID uint64, limit, offset int) ([]model.JobRun, int64, error)
	ListByJobTypeKey(ctx context.Context, typeKey string, limit, offset int) ([]model.JobRunWithJob, int64, error)
	Delete(ctx context.Context, id uint64) error
}

type jobRunRepository struct {
	db *gorm.DB
}

func NewJobRunRepository(db *gorm.DB) JobRunRepository {
	return &jobRunRepository{db: db}
}

func (r *jobRunRepository) Create(ctx context.Context, item *model.JobRun) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *jobRunRepository) GetByID(ctx context.Context, id uint64) (*model.JobRun, error) {
	var item model.JobRun
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *jobRunRepository) GetLatestByJobID(ctx context.Context, jobID uint64) (*model.JobRun, error) {
	var item model.JobRun
	if err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("started_at DESC").
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *jobRunRepository) ListByJobID(ctx context.Context, jobID uint64, limit, offset int) ([]model.JobRun, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.JobRun{}).Where("job_id = ?", jobID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.JobRun
	err := base.Order("started_at DESC").Limit(limit).Offset(offset).Find(&items).Error
	return items, total, err
}

func (r *jobRunRepository) ListByJobTypeKey(ctx context.Context, typeKey string, limit, offset int) ([]model.JobRunWithJob, int64, error) {
	base := r.db.WithContext(ctx).
		Table("job_runs").
		Joins("JOIN jobs ON jobs.id = job_runs.job_id").
		Where("jobs.type_key = ?", typeKey)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.JobRunWithJob
	err := base.Select("job_runs.*, jobs.type_key AS job_type_key, jobs.action_key AS job_action_key").
		Order("job_runs.started_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&items).Error
	return items, total, err
}

func (r *jobRunRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.JobRun{}, id).Error
}
