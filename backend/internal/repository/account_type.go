package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type AccountTypeRepository interface {
    List(ctx context.Context) ([]model.AccountType, error)
    GetByID(ctx context.Context, id uint64) (*model.AccountType, error)
    GetByKey(ctx context.Context, key string) (*model.AccountType, error)
    Create(ctx context.Context, item *model.AccountType) error
    Update(ctx context.Context, item *model.AccountType) error
    DeleteByID(ctx context.Context, id uint64) error
    DeleteByKey(ctx context.Context, key string) error
}

type accountTypeRepository struct {
    db *gorm.DB
}

func NewAccountTypeRepository(db *gorm.DB) AccountTypeRepository {
    return &accountTypeRepository{db: db}
}

func (r *accountTypeRepository) List(ctx context.Context) ([]model.AccountType, error) {
    var items []model.AccountType
    err := r.db.WithContext(ctx).Order("created_at ASC").Find(&items).Error
    return items, err
}

func (r *accountTypeRepository) GetByID(ctx context.Context, id uint64) (*model.AccountType, error) {
    var item model.AccountType
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *accountTypeRepository) GetByKey(ctx context.Context, key string) (*model.AccountType, error) {
    var item model.AccountType
    if err := r.db.WithContext(ctx).Where("key = ?", key).First(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *accountTypeRepository) Create(ctx context.Context, item *model.AccountType) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *accountTypeRepository) Update(ctx context.Context, item *model.AccountType) error {
    return r.db.WithContext(ctx).Save(item).Error
}

func (r *accountTypeRepository) DeleteByID(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.AccountType{}, id).Error
}

func (r *accountTypeRepository) DeleteByKey(ctx context.Context, key string) error {
    return r.db.WithContext(ctx).Where("key = ?", key).Delete(&model.AccountType{}).Error
}
