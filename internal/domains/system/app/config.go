package systemapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func (s Service) GetConfig(ctx context.Context, key string) (json.RawMessage, error) {
	if key == "" {
		return nil, errors.New("key is required")
	}

	var value json.RawMessage
	row := s.db.WithContext(ctx).Raw(
		`SELECT value_json FROM system_configs WHERE key = $1`, key,
	).Row()
	if err := row.Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return json.RawMessage("{}"), nil
		}
		return nil, err
	}
	return value, nil
}

func (s Service) SetConfig(ctx context.Context, key string, value json.RawMessage) error {
	if key == "" {
		return errors.New("key is required")
	}
	if len(value) == 0 || !json.Valid(value) {
		return errors.New("value must be valid JSON")
	}

	result := s.db.WithContext(ctx).Exec(`
		INSERT INTO system_configs (key, value_json, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (key) DO UPDATE
		SET value_json = EXCLUDED.value_json, updated_at = NOW()`,
		key, value,
	)
	return result.Error
}
