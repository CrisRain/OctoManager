package service

import (
	"context"

	"gorm.io/gorm"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/pkg/database"
)

type SystemService interface {
	Status(ctx context.Context) (*dto.SystemStatusResponse, error)
	Migrate(ctx context.Context) (*dto.SystemMigrateResponse, error)
	Setup(ctx context.Context, req *dto.SetupRequest) (*dto.CreateApiKeyResponse, error)
}

type systemService struct {
	db        *gorm.DB
	apiKeySvc ApiKeyService
}

func NewSystemService(db *gorm.DB, apiKeySvc ApiKeyService) SystemService {
	return &systemService{db: db, apiKeySvc: apiKeySvc}
}

func (s *systemService) Status(ctx context.Context) (*dto.SystemStatusResponse, error) {
	hasAdmin, err := s.apiKeySvc.HasAnyAdminKey(ctx)
	if err != nil {
		return nil, internalError("failed to check admin keys", err)
	}
	hasTable := s.db.Migrator().HasTable(&model.ApiKey{})
	return &dto.SystemStatusResponse{
		Initialized: hasTable && hasAdmin,
		NeedsSetup:  !hasAdmin,
	}, nil
}

func (s *systemService) Migrate(ctx context.Context) (*dto.SystemMigrateResponse, error) {
	report, err := database.Migrate(s.db, database.MigrateOptions{})
	if err != nil {
		return nil, internalError("migration failed", err)
	}
	return &dto.SystemMigrateResponse{
		DroppedTables:  report.DroppedTables,
		DroppedColumns: report.DroppedColumns,
	}, nil
}

func (s *systemService) Setup(ctx context.Context, req *dto.SetupRequest) (*dto.CreateApiKeyResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	hasAdmin, err := s.apiKeySvc.HasAnyAdminKey(ctx)
	if err != nil {
		return nil, internalError("failed to check admin keys", err)
	}
	if hasAdmin {
		return nil, conflict("system is already initialized")
	}
	name := trim(req.AdminKeyName)
	if name == "" {
		name = "Admin Key"
	}
	return s.apiKeySvc.Create(ctx, &dto.CreateApiKeyRequest{
		Name: name,
		Role: model.ApiKeyRoleAdmin,
	})
}

var _ SystemService = (*systemService)(nil)
