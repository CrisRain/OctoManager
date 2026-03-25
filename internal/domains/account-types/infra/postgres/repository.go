package accounttypepostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	accounttypedomain "octomanger/internal/domains/account-types/domain"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("account type not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]accounttypedomain.AccountType, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, key, name, category, schema_json, capabilities_json, created_at, updated_at
		FROM account_types
		ORDER BY created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list account types: %w", err)
	}
	defer rows.Close()

	items := make([]accounttypedomain.AccountType, 0)
	for rows.Next() {
		item, err := scanAccountType(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) GetByKey(ctx context.Context, key string) (*accounttypedomain.AccountType, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT id, key, name, category, schema_json, capabilities_json, created_at, updated_at
		FROM account_types
		WHERE key = $1`, key).Row()
	item, err := scanAccountType(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) Create(ctx context.Context, input accounttypedomain.CreateInput) (*accounttypedomain.AccountType, error) {
	schemaJSON, err := json.Marshal(dbutil.NormalizeMap(input.Schema))
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(dbutil.NormalizeMap(input.Capabilities))
	if err != nil {
		return nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO account_types (key, name, category, schema_json, capabilities_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, key, name, category, schema_json, capabilities_json, created_at, updated_at`,
		input.Key, input.Name, input.Category, schemaJSON, capabilitiesJSON,
	).Row()
	item, err := scanAccountType(row)
	if err != nil {
		return nil, fmt.Errorf("create account type: %w", err)
	}
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, key string, input accounttypedomain.PatchInput) (*accounttypedomain.AccountType, error) {
	current, err := r.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.Category != nil {
		current.Category = *input.Category
	}
	if input.Schema != nil {
		current.Schema = input.Schema
	}
	if input.Capabilities != nil {
		current.Capabilities = input.Capabilities
	}

	schemaJSON, err := json.Marshal(dbutil.NormalizeMap(current.Schema))
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(dbutil.NormalizeMap(current.Capabilities))
	if err != nil {
		return nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		UPDATE account_types
		SET name = $2, category = $3, schema_json = $4, capabilities_json = $5, updated_at = NOW()
		WHERE key = $1
		RETURNING id, key, name, category, schema_json, capabilities_json, created_at, updated_at`,
		key, current.Name, current.Category, schemaJSON, capabilitiesJSON,
	).Row()
	item, err := scanAccountType(row)
	if err != nil {
		return nil, fmt.Errorf("patch account type: %w", err)
	}
	return &item, nil
}

// Upsert inserts an account type or updates name/category/schema/capabilities if the key already exists.
func (r Repository) Upsert(ctx context.Context, input accounttypedomain.CreateInput) (*accounttypedomain.AccountType, error) {
	schemaJSON, err := json.Marshal(dbutil.NormalizeMap(input.Schema))
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(dbutil.NormalizeMap(input.Capabilities))
	if err != nil {
		return nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO account_types (key, name, category, schema_json, capabilities_json)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			name              = EXCLUDED.name,
			category          = EXCLUDED.category,
			schema_json       = EXCLUDED.schema_json,
			capabilities_json = EXCLUDED.capabilities_json,
			updated_at        = NOW()
		RETURNING id, key, name, category, schema_json, capabilities_json, created_at, updated_at`,
		input.Key, input.Name, input.Category, schemaJSON, capabilitiesJSON,
	).Row()
	item, err := scanAccountType(row)
	if err != nil {
		return nil, fmt.Errorf("upsert account type: %w", err)
	}
	return &item, nil
}

func (r Repository) Delete(ctx context.Context, key string) error {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM account_types WHERE key = $1`, key)
	if result.Error != nil {
		return fmt.Errorf("delete account type: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAccountType(row scanner) (accounttypedomain.AccountType, error) {
	var item accounttypedomain.AccountType
	var schemaJSON, capabilitiesJSON []byte
	if err := row.Scan(
		&item.ID, &item.Key, &item.Name, &item.Category,
		&schemaJSON, &capabilitiesJSON,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return accounttypedomain.AccountType{}, ErrNotFound
		}
		return accounttypedomain.AccountType{}, fmt.Errorf("scan account type: %w", err)
	}
	item.Schema = dbutil.DecodeJSONMap(schemaJSON)
	item.Capabilities = dbutil.DecodeJSONMap(capabilitiesJSON)
	return item, nil
}

