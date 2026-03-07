package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type TriggerEndpointRepository interface {
    List(ctx context.Context) ([]model.TriggerEndpoint, error)
    GetByID(ctx context.Context, id uint64) (*model.TriggerEndpoint, error)
    GetBySlug(ctx context.Context, slug string) (*model.TriggerEndpoint, error)
    Create(ctx context.Context, item *model.TriggerEndpoint) error
    Update(ctx context.Context, item *model.TriggerEndpoint) error
    Delete(ctx context.Context, id uint64) error
}

type triggerEndpointRepository struct {
    db *gorm.DB
}

func NewTriggerEndpointRepository(db *gorm.DB) TriggerEndpointRepository {
    return &triggerEndpointRepository{db: db}
}

func (r *triggerEndpointRepository) List(ctx context.Context) ([]model.TriggerEndpoint, error) {
    var items []model.TriggerEndpoint
    err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
    return items, err
}

func (r *triggerEndpointRepository) GetByID(ctx context.Context, id uint64) (*model.TriggerEndpoint, error) {
    var item model.TriggerEndpoint
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *triggerEndpointRepository) GetBySlug(ctx context.Context, slug string) (*model.TriggerEndpoint, error) {
    var item model.TriggerEndpoint
    if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *triggerEndpointRepository) Create(ctx context.Context, item *model.TriggerEndpoint) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *triggerEndpointRepository) Update(ctx context.Context, item *model.TriggerEndpoint) error {
    return r.db.WithContext(ctx).Save(item).Error
}

func (r *triggerEndpointRepository) Delete(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.TriggerEndpoint{}, id).Error
}
