package triggerpostgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	triggerdomain "octomanger/internal/domains/triggers/domain"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("trigger not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]triggerdomain.Trigger, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, key, name, job_definition_id, mode, default_input_json, token_prefix, enabled, created_at, updated_at
		FROM triggers ORDER BY created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}
	defer rows.Close()

	items := make([]triggerdomain.Trigger, 0)
	for rows.Next() {
		item, err := scanTrigger(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) GetByID(ctx context.Context, triggerID int64) (*triggerdomain.Trigger, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT id, key, name, job_definition_id, mode, default_input_json, token_prefix, enabled, created_at, updated_at
		FROM triggers WHERE id = $1`, triggerID).Row()
	item, err := scanTrigger(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) GetByKey(ctx context.Context, key string) (*triggerdomain.Trigger, string, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT id, key, name, job_definition_id, mode, default_input_json, token_prefix, enabled, created_at, updated_at, token_hash
		FROM triggers WHERE key = $1`, key).Row()

	var item triggerdomain.Trigger
	var defaultInputJSON []byte
	var tokenHash string
	if err := row.Scan(
		&item.ID, &item.Key, &item.Name, &item.JobDefinitionID, &item.Mode,
		&defaultInputJSON, &item.TokenPrefix, &item.Enabled,
		&item.CreatedAt, &item.UpdatedAt, &tokenHash,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("scan trigger: %w", err)
	}
	item.DefaultInput = dbutil.DecodeJSONMap(defaultInputJSON)
	return &item, tokenHash, nil
}

func (r Repository) Create(ctx context.Context, input triggerdomain.CreateInput, token string) (*triggerdomain.Trigger, error) {
	defaultInputJSON, err := json.Marshal(dbutil.NormalizeMap(input.DefaultInput))
	if err != nil {
		return nil, fmt.Errorf("marshal trigger default input: %w", err)
	}

	tokenHash := hashToken(token)
	tokenPrefix := token
	if len(tokenPrefix) > 8 {
		tokenPrefix = tokenPrefix[:8]
	}

	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO triggers (key, name, job_definition_id, mode, default_input_json, token_hash, token_prefix, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, key, name, job_definition_id, mode, default_input_json, token_prefix, enabled, created_at, updated_at`,
		input.Key, input.Name, input.JobDefinitionID, input.Mode,
		defaultInputJSON, tokenHash, tokenPrefix, input.Enabled,
	).Row()
	item, err := scanTrigger(row)
	if err != nil {
		return nil, fmt.Errorf("create trigger: %w", err)
	}
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, triggerID int64, input triggerdomain.PatchTriggerInput) (*triggerdomain.Trigger, error) {
	current, err := r.GetByID(ctx, triggerID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.Mode != nil {
		current.Mode = *input.Mode
	}
	if input.Enabled != nil {
		current.Enabled = *input.Enabled
	}
	row := r.db.WithContext(ctx).Raw(`
		UPDATE triggers SET name = $2, mode = $3, enabled = $4, updated_at = NOW() WHERE id = $1
		RETURNING id, key, name, job_definition_id, mode, default_input_json, token_prefix, enabled, created_at, updated_at`,
		triggerID, current.Name, current.Mode, current.Enabled,
	).Row()
	item, err := scanTrigger(row)
	if err != nil {
		return nil, fmt.Errorf("patch trigger: %w", err)
	}
	return &item, nil
}

func (r Repository) Delete(ctx context.Context, triggerID int64) error {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM triggers WHERE id = $1`, triggerID)
	if result.Error != nil {
		return fmt.Errorf("delete trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func VerifyToken(token, hash string) bool {
	return hashToken(token) == hash
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTrigger(row scanner) (triggerdomain.Trigger, error) {
	var item triggerdomain.Trigger
	var defaultInputJSON []byte
	if err := row.Scan(
		&item.ID, &item.Key, &item.Name, &item.JobDefinitionID, &item.Mode,
		&defaultInputJSON, &item.TokenPrefix, &item.Enabled,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return triggerdomain.Trigger{}, ErrNotFound
		}
		return triggerdomain.Trigger{}, fmt.Errorf("scan trigger: %w", err)
	}
	item.DefaultInput = dbutil.DecodeJSONMap(defaultInputJSON)
	return item, nil
}

