package repository

import (
    "context"
    "time"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type ApiKeyRepository interface {
    List(ctx context.Context) ([]model.ApiKey, error)
    GetByID(ctx context.Context, id uint64) (*model.ApiKey, error)
    GetByHash(ctx context.Context, hash string) (*model.ApiKey, error)
    Create(ctx context.Context, item *model.ApiKey) error
    Update(ctx context.Context, item *model.ApiKey) error
    UpdateEnabled(ctx context.Context, id uint64, enabled bool) (*model.ApiKey, error)
    UpdateLastUsed(ctx context.Context, id uint64) error
    Delete(ctx context.Context, id uint64) error
}

type apiKeyRepository struct {
    db *gorm.DB
}

func NewApiKeyRepository(db *gorm.DB) ApiKeyRepository {
    return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) List(ctx context.Context) ([]model.ApiKey, error) {
    var items []model.ApiKey
    err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
    return items, err
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uint64) (*model.ApiKey, error) {
    var item model.ApiKey
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *apiKeyRepository) GetByHash(ctx context.Context, hash string) (*model.ApiKey, error) {
    var item model.ApiKey
    if err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *apiKeyRepository) Create(ctx context.Context, item *model.ApiKey) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *apiKeyRepository) Update(ctx context.Context, item *model.ApiKey) error {
    return r.db.WithContext(ctx).Save(item).Error
}

func (r *apiKeyRepository) UpdateEnabled(ctx context.Context, id uint64, enabled bool) (*model.ApiKey, error) {
    if err := r.db.WithContext(ctx).
        Model(&model.ApiKey{}).
        Where("id = ?", id).
        Updates(map[string]any{"enabled": enabled, "updated_at": time.Now().UTC()}).Error; err != nil {
        return nil, err
    }
    return r.GetByID(ctx, id)
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).
        Model(&model.ApiKey{}).
        Where("id = ?", id).
        Updates(map[string]any{"last_used_at": time.Now().UTC(), "updated_at": time.Now().UTC()}).Error
}

func (r *apiKeyRepository) Delete(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.ApiKey{}, id).Error
}
