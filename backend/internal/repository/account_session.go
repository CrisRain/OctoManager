package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type AccountSessionRepository interface {
    Create(ctx context.Context, item *model.AccountSession) error
    GetByID(ctx context.Context, id uint64) (*model.AccountSession, error)
    ListByAccountID(ctx context.Context, accountID uint64, limit, offset int) ([]model.AccountSession, int64, error)
    Delete(ctx context.Context, id uint64) error
}

type accountSessionRepository struct {
    db *gorm.DB
}

func NewAccountSessionRepository(db *gorm.DB) AccountSessionRepository {
    return &accountSessionRepository{db: db}
}

func (r *accountSessionRepository) Create(ctx context.Context, item *model.AccountSession) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *accountSessionRepository) GetByID(ctx context.Context, id uint64) (*model.AccountSession, error) {
    var item model.AccountSession
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *accountSessionRepository) ListByAccountID(ctx context.Context, accountID uint64, limit, offset int) ([]model.AccountSession, int64, error) {
    base := r.db.WithContext(ctx).Model(&model.AccountSession{}).Where("account_id = ?", accountID)

    var total int64
    if err := base.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    var items []model.AccountSession
    err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error
    return items, total, err
}

func (r *accountSessionRepository) Delete(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.AccountSession{}, id).Error
}
