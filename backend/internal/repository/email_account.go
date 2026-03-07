package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type EmailAccountRepository interface {
    List(ctx context.Context) ([]model.EmailAccount, error)
    ListPaged(ctx context.Context, limit, offset int) ([]model.EmailAccount, int64, error)
    GetByID(ctx context.Context, id uint64) (*model.EmailAccount, error)
    GetByAddress(ctx context.Context, address string) (*model.EmailAccount, error)
    Create(ctx context.Context, item *model.EmailAccount) error
    Update(ctx context.Context, item *model.EmailAccount) error
    Delete(ctx context.Context, id uint64) error
}

type emailAccountRepository struct {
    db *gorm.DB
}

func NewEmailAccountRepository(db *gorm.DB) EmailAccountRepository {
    return &emailAccountRepository{db: db}
}

func (r *emailAccountRepository) List(ctx context.Context) ([]model.EmailAccount, error) {
    var items []model.EmailAccount
    err := r.db.WithContext(ctx).Order("created_at ASC").Find(&items).Error
    return items, err
}

func (r *emailAccountRepository) ListPaged(ctx context.Context, limit, offset int) ([]model.EmailAccount, int64, error) {
    base := r.db.WithContext(ctx).Model(&model.EmailAccount{})

    var total int64
    if err := base.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    var items []model.EmailAccount
    err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error
    return items, total, err
}

func (r *emailAccountRepository) GetByID(ctx context.Context, id uint64) (*model.EmailAccount, error) {
    var item model.EmailAccount
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *emailAccountRepository) GetByAddress(ctx context.Context, address string) (*model.EmailAccount, error) {
    var item model.EmailAccount
    if err := r.db.WithContext(ctx).Where("address = ?", address).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *emailAccountRepository) Create(ctx context.Context, item *model.EmailAccount) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *emailAccountRepository) Update(ctx context.Context, item *model.EmailAccount) error {
    return r.db.WithContext(ctx).Save(item).Error
}

func (r *emailAccountRepository) Delete(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.EmailAccount{}, id).Error
}
