package service

import (
	"context"
	"encoding/json"
	"fmt"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/octomodule"
	"octomanger/backend/internal/repository"
)

type AccountTypeService interface {
	List(ctx context.Context) ([]dto.AccountTypeResponse, error)
	Get(ctx context.Context, key string) (*dto.AccountTypeResponse, error)
	Create(ctx context.Context, req *dto.CreateAccountTypeRequest) (*dto.AccountTypeResponse, error)
	Patch(ctx context.Context, key string, req *dto.PatchAccountTypeRequest) (*dto.AccountTypeResponse, error)
	Delete(ctx context.Context, key string) error
}

type accountTypeService struct {
	repo      repository.AccountTypeRepository
	moduleDir string
}

func NewAccountTypeService(repo repository.AccountTypeRepository, moduleDir string) AccountTypeService {
	return &accountTypeService{
		repo:      repo,
		moduleDir: moduleDir,
	}
}

func (s *accountTypeService) List(ctx context.Context) ([]dto.AccountTypeResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list account types", err)
	}
	responses := make([]dto.AccountTypeResponse, 0, len(items))
	for i := range items {
		if response := accountTypeToResponse(&items[i]); response != nil {
			responses = append(responses, *response)
		}
	}
	return responses, nil
}

func (s *accountTypeService) Get(ctx context.Context, key string) (*dto.AccountTypeResponse, error) {
	trimmed := trim(key)
	if trimmed == "" {
		return nil, invalidInput("account type key is required")
	}

	item, err := s.repo.GetByKey(ctx, trimmed)
	if err != nil {
		return nil, wrapRepoError(err, "account type not found")
	}
	return accountTypeToResponse(item), nil
}

func (s *accountTypeService) Create(ctx context.Context, req *dto.CreateAccountTypeRequest) (*dto.AccountTypeResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	key := trim(req.Key)
	name := trim(req.Name)
	category := trim(req.Category)
	if key == "" {
		return nil, invalidInput("key is required")
	}
	if name == "" {
		return nil, invalidInput("name is required")
	}
	if category == "" {
		return nil, invalidInput("category is required")
	}
	if !isValidCategory(category) {
		return nil, invalidInput("category must be one of: system, email, generic")
	}
	if !isJSONObject(req.Schema) {
		return nil, invalidInput("schema must be a valid JSON object")
	}
	if !isJSONObject(req.Capabilities) {
		return nil, invalidInput("capabilities must be a valid JSON object")
	}
	if len(req.ScriptConfig) > 0 && !isJSONObjectOrNull(req.ScriptConfig) {
		return nil, invalidInput("script_config must be a valid JSON object or null")
	}

	scriptConfig := normalizeJSON(req.ScriptConfig, "null")
	if isGenericCategory(category) {
		resolved, err := octomodule.ResolveEntryPath(s.moduleDir, key, scriptConfig)
		if err != nil {
			return nil, invalidInput(fmt.Sprintf("invalid script_config: %s", err.Error()))
		}
		if _, err := octomodule.EnsureScriptFile(resolved.EntryPath, key); err != nil {
			return nil, internalError("failed to initialize octoModule script", err)
		}
	}

	item := &model.AccountType{
		Key:          key,
		Name:         name,
		Category:     category,
		Schema:       normalizeJSON(req.Schema, "{}"),
		Capabilities: normalizeJSON(req.Capabilities, "{}"),
		ScriptConfig: scriptConfig,
		Version:      1,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("account type key already exists")
		}
		return nil, internalError("failed to create account type", err)
	}
	return accountTypeToResponse(item), nil
}

func (s *accountTypeService) Patch(ctx context.Context, key string, req *dto.PatchAccountTypeRequest) (*dto.AccountTypeResponse, error) {
	trimmedKey := trim(key)
	if trimmedKey == "" {
		return nil, invalidInput("account type key is required")
	}
	if req == nil {
		return nil, invalidInput("payload is required")
	}

	current, err := s.repo.GetByKey(ctx, trimmedKey)
	if err != nil {
		return nil, wrapRepoError(err, "account type not found")
	}

	hasChanges := false
	if req.Name != nil {
		hasChanges = true
		trimmed := trim(*req.Name)
		if trimmed == "" {
			return nil, invalidInput("name cannot be empty")
		}
		current.Name = trimmed
	}
	if req.Category != nil {
		hasChanges = true
		trimmed := trim(*req.Category)
		if trimmed == "" {
			return nil, invalidInput("category cannot be empty")
		}
		if !isValidCategory(trimmed) {
			return nil, invalidInput("category must be one of: system, email, generic")
		}
		current.Category = trimmed
	}
	if req.Schema != nil {
		hasChanges = true
		if !isJSONObject(*req.Schema) {
			return nil, invalidInput("schema must be a valid JSON object")
		}
		current.Schema = normalizeJSON(*req.Schema, "{}")
	}
	if req.Capabilities != nil {
		hasChanges = true
		if !isJSONObject(*req.Capabilities) {
			return nil, invalidInput("capabilities must be a valid JSON object")
		}
		current.Capabilities = normalizeJSON(*req.Capabilities, "{}")
	}
	if req.ScriptConfig != nil {
		hasChanges = true
		if !isJSONObjectOrNull(*req.ScriptConfig) {
			return nil, invalidInput("script_config must be a valid JSON object or null")
		}
		current.ScriptConfig = normalizeJSON(*req.ScriptConfig, "null")
	}

	if !hasChanges {
		return nil, invalidInput("at least one field is required")
	}

	if isGenericCategory(current.Category) {
		resolved, err := octomodule.ResolveEntryPath(s.moduleDir, trimmedKey, current.ScriptConfig)
		if err != nil {
			return nil, invalidInput(fmt.Sprintf("invalid script_config: %s", err.Error()))
		}
		if _, err := octomodule.EnsureScriptFile(resolved.EntryPath, trimmedKey); err != nil {
			return nil, internalError("failed to initialize octoModule script", err)
		}
	}

	current.Version++
	if err := s.repo.Update(ctx, current); err != nil {
		if isDuplicateKeyError(err) {
			return nil, conflict("account type key already exists")
		}
		return nil, internalError("failed to update account type", err)
	}
	return accountTypeToResponse(current), nil
}

func (s *accountTypeService) Delete(ctx context.Context, key string) error {
	trimmed := trim(key)
	if trimmed == "" {
		return invalidInput("account type key is required")
	}

	if _, err := s.repo.GetByKey(ctx, trimmed); err != nil {
		return wrapRepoError(err, "account type not found")
	}
	if err := s.repo.DeleteByKey(ctx, trimmed); err != nil {
		if isForeignKeyViolation(err) {
			return conflict("account type is still in use")
		}
		return internalError("failed to delete account type", err)
	}
	return nil
}

func accountTypeToResponse(item *model.AccountType) *dto.AccountTypeResponse {
	if item == nil {
		return nil
	}

	response := dto.AccountTypeResponse{
		ID:           item.ID,
		Key:          item.Key,
		Name:         item.Name,
		Category:     item.Category,
		Schema:       normalizeJSON(item.Schema, "{}"),
		Capabilities: normalizeJSON(item.Capabilities, "{}"),
		Version:      item.Version,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}

	if len(item.ScriptConfig) > 0 && !json.Valid(item.ScriptConfig) {
		response.ScriptConfig = json.RawMessage("null")
	} else if len(item.ScriptConfig) > 0 {
		response.ScriptConfig = item.ScriptConfig
	}

	return &response
}
