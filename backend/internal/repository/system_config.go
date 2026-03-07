package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

type SystemConfigRepository interface {
    List(ctx context.Context) ([]model.SystemConfig, error)
    GetByKey(ctx context.Context, key string) (*model.SystemConfig, error)
    Upsert(ctx context.Context, item *model.SystemConfig) error
    Delete(ctx context.Context, key string) error
}

type systemConfigRepository struct {
    db *gorm.DB
}

func NewSystemConfigRepository(db *gorm.DB) SystemConfigRepository {
    return &systemConfigRepository{db: db}
}

func (r *systemConfigRepository) List(ctx context.Context) ([]model.SystemConfig, error) {
    var items []model.SystemConfig
    err := r.db.WithContext(ctx).Order("created_at ASC").Find(&items).Error
    return items, err
}

func (r *systemConfigRepository) GetByKey(ctx context.Context, key string) (*model.SystemConfig, error) {
    var item model.SystemConfig
    if err := r.db.WithContext(ctx).Where("key = ?", key).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *systemConfigRepository) Upsert(ctx context.Context, item *model.SystemConfig) error {
    return r.db.WithContext(ctx).Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "key"}},
        DoUpdates: clause.AssignmentColumns([]string{"value", "is_critical", "description", "updated_at"}),
    }).Create(item).Error
}

func (r *systemConfigRepository) Delete(ctx context.Context, key string) error {
    return r.db.WithContext(ctx).Where("key = ?", key).Delete(&model.SystemConfig{}).Error
}
