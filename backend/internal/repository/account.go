package repository

import (
    "context"

    "octomanger/backend/internal/model"
    "gorm.io/gorm"
)

type AccountRepository interface {
	List(ctx context.Context) ([]model.Account, error)
	ListPaged(ctx context.Context, limit, offset int, typeKey string) ([]model.Account, int64, error)
	ListByTypeKey(ctx context.Context, typeKey string) ([]model.Account, error)
	GetByID(ctx context.Context, id uint64) (*model.Account, error)
	Create(ctx context.Context, item *model.Account) error
	Update(ctx context.Context, item *model.Account) error
	Delete(ctx context.Context, id uint64) error
}

type accountRepository struct {
    db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
    return &accountRepository{db: db}
}

func (r *accountRepository) List(ctx context.Context) ([]model.Account, error) {
    var items []model.Account
    err := r.db.WithContext(ctx).Order("created_at ASC").Find(&items).Error
    return items, err
}

func (r *accountRepository) ListPaged(ctx context.Context, limit, offset int, typeKey string) ([]model.Account, int64, error) {
    base := r.db.WithContext(ctx).Model(&model.Account{})
    if typeKey != "" {
        base = base.Where("type_key = ?", typeKey)
    }

    var total int64
    if err := base.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    var items []model.Account
    err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error
    return items, total, err
}

func (r *accountRepository) ListByTypeKey(ctx context.Context, typeKey string) ([]model.Account, error) {
	var items []model.Account
	query := r.db.WithContext(ctx).Model(&model.Account{})
	if typeKey != "" {
		query = query.Where("type_key = ?", typeKey)
	}
	err := query.Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *accountRepository) GetByID(ctx context.Context, id uint64) (*model.Account, error) {
    var item model.Account
    if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (r *accountRepository) Create(ctx context.Context, item *model.Account) error {
    return r.db.WithContext(ctx).Create(item).Error
}

func (r *accountRepository) Update(ctx context.Context, item *model.Account) error {
    return r.db.WithContext(ctx).Save(item).Error
}

func (r *accountRepository) Delete(ctx context.Context, id uint64) error {
    return r.db.WithContext(ctx).Delete(&model.Account{}, id).Error
}
