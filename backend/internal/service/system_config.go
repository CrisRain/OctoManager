package service

import (
	"context"
	"encoding/json"

	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
)

type SystemConfigService interface {
	Get(ctx context.Context, key string) (json.RawMessage, error)
	Set(ctx context.Context, key string, value json.RawMessage) error
}

type systemConfigService struct {
	repo repository.SystemConfigRepository
}

func NewSystemConfigService(repo repository.SystemConfigRepository) SystemConfigService {
	return &systemConfigService{repo: repo}
}

func (s *systemConfigService) Get(ctx context.Context, key string) (json.RawMessage, error) {
	item, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return nil, notFound("config not found")
		}
		return nil, internalError("failed to get config", err)
	}
	return item.Value, nil
}

func (s *systemConfigService) Set(ctx context.Context, key string, value json.RawMessage) error {
	if key == "" {
		return invalidInput("key is required")
	}
	if len(value) == 0 || !json.Valid(value) {
		return invalidInput("value must be valid JSON")
	}
	item := &model.SystemConfig{
		Key:   key,
		Value: value,
	}
	if err := s.repo.Upsert(ctx, item); err != nil {
		return internalError("failed to save config", err)
	}
	return nil
}

var _ SystemConfigService = (*systemConfigService)(nil)
