package accountpostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	accountdomain "octomanger/internal/domains/accounts/domain"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("account not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]accountdomain.Account, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT a.id, a.account_type_id, COALESCE(t.key, ''), a.identifier, a.status, a.tags_json, a.spec_json, a.created_at, a.updated_at
		FROM accounts a
		LEFT JOIN account_types t ON t.id = a.account_type_id
		ORDER BY a.created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	items := make([]accountdomain.Account, 0)
	for rows.Next() {
		item, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) Get(ctx context.Context, accountID int64) (*accountdomain.Account, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT a.id, a.account_type_id, COALESCE(t.key, ''), a.identifier, a.status, a.tags_json, a.spec_json, a.created_at, a.updated_at
		FROM accounts a
		LEFT JOIN account_types t ON t.id = a.account_type_id
		WHERE a.id = $1`, accountID).Row()
	item, err := scanAccount(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) Create(ctx context.Context, input accountdomain.CreateInput) (*accountdomain.Account, error) {
	specJSON, err := json.Marshal(dbutil.NormalizeMap(input.Spec))
	if err != nil {
		return nil, fmt.Errorf("marshal account spec: %w", err)
	}
	tagsJSON, err := json.Marshal(normalizeStrings(input.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal account tags: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		WITH ins AS (
			INSERT INTO accounts (account_type_id, identifier, status, tags_json, spec_json)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, account_type_id, identifier, status, tags_json, spec_json, created_at, updated_at
		)
		SELECT i.id, i.account_type_id, COALESCE(t.key, ''), i.identifier, i.status,
		       i.tags_json, i.spec_json, i.created_at, i.updated_at
		FROM ins i
		LEFT JOIN account_types t ON t.id = i.account_type_id`,
		input.AccountTypeID, input.Identifier, input.Status, tagsJSON, specJSON,
	).Row()
	item, err := scanAccount(row)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, accountID int64, input accountdomain.PatchInput) (*accountdomain.Account, error) {
	current, err := r.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if input.Status != nil {
		current.Status = *input.Status
	}
	if input.Tags != nil {
		current.Tags = input.Tags
	}
	if input.Spec != nil {
		current.Spec = input.Spec
	}

	specJSON, err := json.Marshal(dbutil.NormalizeMap(current.Spec))
	if err != nil {
		return nil, fmt.Errorf("marshal account spec: %w", err)
	}
	tagsJSON, err := json.Marshal(normalizeStrings(current.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal account tags: %w", err)
	}

	row := r.db.WithContext(ctx).Raw(`
		WITH upd AS (
			UPDATE accounts
			SET status = $2, tags_json = $3, spec_json = $4, updated_at = NOW()
			WHERE id = $1
			RETURNING id, account_type_id, identifier, status, tags_json, spec_json, created_at, updated_at
		)
		SELECT u.id, u.account_type_id, COALESCE(t.key, ''), u.identifier, u.status,
		       u.tags_json, u.spec_json, u.created_at, u.updated_at
		FROM upd u
		LEFT JOIN account_types t ON t.id = u.account_type_id`,
		accountID, current.Status, tagsJSON, specJSON,
	).Row()
	item, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("patch account: %w", err)
	}
	return &item, nil
}

func (r Repository) Delete(ctx context.Context, accountID int64) error {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM accounts WHERE id = $1`, accountID)
	if result.Error != nil {
		return fmt.Errorf("delete account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAccount(row scanner) (accountdomain.Account, error) {
	var item accountdomain.Account
	var accountTypeID sql.NullInt64
	var tagsJSON, specJSON []byte
	if err := row.Scan(
		&item.ID, &accountTypeID, &item.AccountTypeKey,
		&item.Identifier, &item.Status,
		&tagsJSON, &specJSON,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return accountdomain.Account{}, ErrNotFound
		}
		return accountdomain.Account{}, fmt.Errorf("scan account: %w", err)
	}
	if accountTypeID.Valid {
		v := accountTypeID.Int64
		item.AccountTypeID = &v
	}
	item.Tags = decodeJSONStringArray(tagsJSON)
	item.Spec = dbutil.DecodeJSONMap(specJSON)
	return item, nil
}

func decodeJSONStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	v := []string{}
	_ = json.Unmarshal(raw, &v)
	return v
}

func normalizeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
