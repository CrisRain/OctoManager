package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type JobRepository interface {
    List(ctx context.Context) ([]model.Job, error)
    ListPaged(ctx context.Context, limit, offset int) ([]model.Job, int64, error)
    GetByID(ctx context.Context, id uint64) (*model.Job, error)
    Create(ctx context.Context, item *model.Job) error
    Update(ctx context.Context, item *model.Job) error
    UpdateStatus(ctx context.Context, id uint64, status int16) (*model.Job, error)
    Delete(ctx context.Context, id uint64) error
}

type jobRepository struct {
    db *gorm.DB
}

func NewJobRepository(db *gorm.DB) JobRepository {
    return &jobRepository{db: db}
}

func (r *jobRepository) List(ctx context.Context) ([]model.Job, error) {
    var items []model.Job
    err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
    return items, err
}

func (r *jobRepository) ListPaged(ctx context.Context, limit, offset int) ([]model.Job, int64, error) {
    base := r.db.WithContext(ctx).Model(&model.Job{})

    var total int64
    if err := base.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    var items []model.Job
    err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error
    return items, total, err
}

func (r *jobRepository) GetByID(ctx context.Context, id uint64) (*model.Job, error) {
    var item model.Job
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *jobRepository) Create(ctx context.Context, item *model.Job) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *jobRepository) Update(ctx context.Context, item *model.Job) error {
    return r.db.WithContext(ctx).Save(item).Error
}

func (r *jobRepository) UpdateStatus(ctx context.Context, id uint64, status int16) (*model.Job, error) {
    if err := r.db.WithContext(ctx).
        Model(&model.Job{}).
        Where("id = ?", id).
        Update("status", status).Error; err != nil {
        return nil, err
    }
    return r.GetByID(ctx, id)
}

func (r *jobRepository) Delete(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.Job{}, id).Error
}
