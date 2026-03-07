package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/octomodule"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/worker/bridge"
)

type OctoModuleRunner interface {
	ExecuteWithScript(ctx context.Context, scriptPath string, input bridge.Input) (bridge.Output, error)
}

type OctoModuleService interface {
	List(ctx context.Context) ([]dto.OctoModuleInfoResponse, error)
	Get(ctx context.Context, typeKey string) (*dto.OctoModuleInfoResponse, error)
	DryRun(ctx context.Context, typeKey string, req *dto.OctoModuleDryRunRequest) (*dto.OctoModuleDryRunResponse, error)
	EnsureByTypeKey(ctx context.Context, typeKey string) (*dto.OctoModuleEnsureResponse, error)
	SyncMissing(ctx context.Context) (*dto.OctoModuleSyncResponse, error)
	GetScript(ctx context.Context, typeKey string) (*dto.OctoModuleScriptResponse, error)
	UpdateScript(ctx context.Context, typeKey string, req *dto.UpdateOctoModuleScriptRequest) error
	ListFiles(ctx context.Context, typeKey string) (*dto.ListOctoModuleFilesResponse, error)
	GetFile(ctx context.Context, typeKey, filename string) (*dto.OctoModuleScriptResponse, error)
	UpdateFile(ctx context.Context, typeKey, filename string, req *dto.UpdateOctoModuleFileRequest) error
	GetRunHistory(ctx context.Context, typeKey string, limit, offset int) (*dto.OctoModuleRunHistoryResponse, error)
	GetVenvInfo(ctx context.Context, typeKey string) (*dto.VenvInfoResponse, error)
	InstallDeps(ctx context.Context, typeKey string, req *dto.InstallDepsRequest) (*dto.InstallDepsResponse, error)
}

type octoModuleService struct {
	accountTypeRepo repository.AccountTypeRepository
	jobRunRepo      repository.JobRunRepository
	runner          OctoModuleRunner
	moduleDir       string
	pythonBin       string
}

func NewOctoModuleService(
	accountTypeRepo repository.AccountTypeRepository,
	jobRunRepo repository.JobRunRepository,
	runner OctoModuleRunner,
	moduleDir string,
	pythonBin string,
) OctoModuleService {
	return &octoModuleService{
		accountTypeRepo: accountTypeRepo,
		jobRunRepo:      jobRunRepo,
		runner:          runner,
		moduleDir:       moduleDir,
		pythonBin:       pythonBin,
	}
}

func (s *octoModuleService) List(ctx context.Context) ([]dto.OctoModuleInfoResponse, error) {
	items, err := s.accountTypeRepo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list account types", err)
	}
	responses := make([]dto.OctoModuleInfoResponse, 0, len(items))
	for i := range items {
		if !isGenericCategory(items[i].Category) {
			continue
		}
		responses = append(responses, s.buildInfo(&items[i]))
	}
	return responses, nil
}

func (s *octoModuleService) Get(ctx context.Context, typeKey string) (*dto.OctoModuleInfoResponse, error) {
	item, err := s.getGenericAccountType(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	response := s.buildInfo(item)
	return &response, nil
}

func (s *octoModuleService) DryRun(
	ctx context.Context,
	typeKey string,
	req *dto.OctoModuleDryRunRequest,
) (*dto.OctoModuleDryRunResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}

	action := trim(req.Action)
	identifier := trim(req.Account.Identifier)
	if action == "" {
		return nil, invalidInput("action is required")
	}
	if identifier == "" {
		return nil, invalidInput("account.identifier is required")
	}

	if info.ScriptPath == "" {
		return nil, invalidInput("module script path is empty")
	}
	if !info.Exists {
		return nil, invalidInput("module script does not exist")
	}
	if s.runner == nil {
		return nil, internalError("octoModule runner is not configured", errors.New("missing runner"))
	}

	spec := req.Account.Spec
	if spec == nil {
		spec = map[string]any{}
	}
	params := req.Params
	if params == nil {
		params = map[string]any{}
	}

	requestID := trim(req.Context.RequestID)
	if requestID == "" {
		requestID = fmt.Sprintf("dry-run:%s", trim(typeKey))
	}

	output, runErr := s.runner.ExecuteWithScript(ctx, info.ScriptPath, bridge.Input{
		Action: action,
		Account: bridge.InputAccount{
			Identifier: identifier,
			Spec:       spec,
		},
		Params: params,
		Context: bridge.InputContext{
			RequestID: requestID,
		},
	})
	if runErr != nil {
		return nil, internalError("failed to execute octoModule dry-run", runErr)
	}

	return &dto.OctoModuleDryRunResponse{
		Module: *info,
		Output: dto.OctoModuleOutputResponse{
			Status:       output.Status,
			Result:       output.Result,
			ErrorCode:    output.ErrorCode,
			ErrorMessage: output.ErrorMessage,
		},
	}, nil
}

func (s *octoModuleService) EnsureByTypeKey(ctx context.Context, typeKey string) (*dto.OctoModuleEnsureResponse, error) {
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	if info.Error != "" {
		return nil, invalidInput("failed to resolve module script path: " + info.Error)
	}

	created, ensureErr := octomodule.EnsureScriptFile(info.ScriptPath, info.TypeKey)
	if ensureErr != nil {
		return nil, internalError("failed to ensure octoModule script", ensureErr)
	}
	info.Exists = octomodule.FileExists(info.ScriptPath)

	return &dto.OctoModuleEnsureResponse{
		Module:  *info,
		Created: created,
	}, nil
}

func (s *octoModuleService) SyncMissing(ctx context.Context) (*dto.OctoModuleSyncResponse, error) {
	items, err := s.accountTypeRepo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list account types", err)
	}

	genericItems := make([]model.AccountType, 0, len(items))
	for i := range items {
		if isGenericCategory(items[i].Category) {
			genericItems = append(genericItems, items[i])
		}
	}

	result := dto.OctoModuleSyncResponse{
		Total: len(genericItems),
		Items: make([]dto.OctoModuleSyncItemResponse, 0, len(genericItems)),
	}

	for i := range genericItems {
		info := s.buildInfo(&genericItems[i])
		item := dto.OctoModuleSyncItemResponse{
			TypeKey:    info.TypeKey,
			Category:   info.Category,
			ScriptPath: info.ScriptPath,
			Source:     info.Source,
			Exists:     info.Exists,
			Config:     info.ScriptConfig,
		}

		if info.Error != "" {
			item.Error = info.Error
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		created, err := octomodule.EnsureScriptFile(info.ScriptPath, info.TypeKey)
		if err != nil {
			item.Error = err.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		item.Created = created
		item.Exists = octomodule.FileExists(info.ScriptPath)
		if created {
			result.Created++
		} else {
			result.Existing++
		}
		result.Items = append(result.Items, item)
	}

	return &result, nil
}

func (s *octoModuleService) GetScript(ctx context.Context, typeKey string) (*dto.OctoModuleScriptResponse, error) {
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	if info.ScriptPath == "" {
		return nil, invalidInput("module script path is empty")
	}
	content, err := os.ReadFile(info.ScriptPath)
	if err != nil {
		return nil, internalError("failed to read script file", err)
	}
	return &dto.OctoModuleScriptResponse{Content: string(content)}, nil
}

func (s *octoModuleService) UpdateScript(ctx context.Context, typeKey string, req *dto.UpdateOctoModuleScriptRequest) error {
	if req == nil {
		return invalidInput("payload is required")
	}
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return err
	}
	if info.ScriptPath == "" {
		return invalidInput("module script path is empty")
	}
	if err := os.WriteFile(info.ScriptPath, []byte(req.Content), 0644); err != nil {
		return internalError("failed to write script file", err)
	}
	return nil
}

func (s *octoModuleService) GetRunHistory(
	ctx context.Context,
	typeKey string,
	limit, offset int,
) (*dto.OctoModuleRunHistoryResponse, error) {
	item, err := s.getGenericAccountType(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	items, total, err := s.jobRunRepo.ListByJobTypeKey(ctx, item.Key, limit, offset)
	if err != nil {
		return nil, internalError("failed to list job runs", err)
	}
	if items == nil {
		items = []model.JobRunWithJob{}
	}

	responses := make([]dto.JobRunResponse, 0, len(items))
	for i := range items {
		responses = append(responses, jobRunToResponse(items[i]))
	}

	return &dto.OctoModuleRunHistoryResponse{
		Items:  responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *octoModuleService) getGenericAccountType(ctx context.Context, typeKey string) (*model.AccountType, error) {
	trimmed := trim(typeKey)
	if trimmed == "" {
		return nil, invalidInput("type_key is required")
	}
	item, err := s.accountTypeRepo.GetByKey(ctx, trimmed)
	if err != nil {
		return nil, wrapRepoError(err, "octo module not found")
	}
	if !isGenericCategory(item.Category) {
		return nil, notFound("octo module not found")
	}
	return item, nil
}

func (s *octoModuleService) ListFiles(ctx context.Context, typeKey string) (*dto.ListOctoModuleFilesResponse, error) {
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	if info.ModuleDir == "" {
		return nil, invalidInput("module directory could not be resolved")
	}
	entries, err := os.ReadDir(info.ModuleDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &dto.ListOctoModuleFilesResponse{
				ModuleDir: info.ModuleDir,
				EntryFile: info.EntryFile,
				Files:     []dto.OctoModuleFileInfo{},
			}, nil
		}
		return nil, internalError("failed to read module directory", err)
	}
	files := make([]dto.OctoModuleFileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, dto.OctoModuleFileInfo{
			Name:    e.Name(),
			Size:    fi.Size(),
			IsEntry: e.Name() == info.EntryFile,
		})
	}
	return &dto.ListOctoModuleFilesResponse{
		ModuleDir: info.ModuleDir,
		EntryFile: info.EntryFile,
		Files:     files,
	}, nil
}

func (s *octoModuleService) GetFile(ctx context.Context, typeKey, filename string) (*dto.OctoModuleScriptResponse, error) {
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	if info.ModuleDir == "" {
		return nil, invalidInput("module directory could not be resolved")
	}
	absPath, err := octomodule.ResolveFileInModuleDir(info.ModuleDir, filename)
	if err != nil {
		return nil, invalidInput(err.Error())
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, internalError("failed to read file", err)
	}
	return &dto.OctoModuleScriptResponse{Content: string(content)}, nil
}

func (s *octoModuleService) UpdateFile(ctx context.Context, typeKey, filename string, req *dto.UpdateOctoModuleFileRequest) error {
	if req == nil {
		return invalidInput("payload is required")
	}
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return err
	}
	if info.ModuleDir == "" {
		return invalidInput("module directory could not be resolved")
	}
	absPath, err := octomodule.ResolveFileInModuleDir(info.ModuleDir, filename)
	if err != nil {
		return invalidInput(err.Error())
	}
	if mkErr := os.MkdirAll(filepath.Dir(absPath), 0o755); mkErr != nil {
		return internalError("failed to create directory", mkErr)
	}
	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		return internalError("failed to write file", err)
	}
	return nil
}

func (s *octoModuleService) buildInfo(item *model.AccountType) dto.OctoModuleInfoResponse {
	if item == nil {
		return dto.OctoModuleInfoResponse{}
	}
	resolved, err := octomodule.ResolveEntryPath(s.moduleDir, item.Key, item.ScriptConfig)
	if err != nil {
		return dto.OctoModuleInfoResponse{
			TypeKey:      item.Key,
			Category:     item.Category,
			ScriptConfig: item.ScriptConfig,
			Error:        err.Error(),
		}
	}
	return dto.OctoModuleInfoResponse{
		TypeKey:      item.Key,
		Category:     item.Category,
		ScriptPath:   resolved.EntryPath,
		ModuleDir:    filepath.Dir(resolved.EntryPath),
		EntryFile:    filepath.Base(resolved.EntryPath),
		Source:       resolved.Source,
		Exists:       octomodule.FileExists(resolved.EntryPath),
		ScriptConfig: item.ScriptConfig,
	}
}

func jobRunToResponse(item model.JobRunWithJob) dto.JobRunResponse {
	return dto.JobRunResponse{
		ID:           item.ID,
		JobID:        item.JobID,
		JobTypeKey:   item.JobTypeKey,
		JobActionKey: item.JobActionKey,
		AccountID:    item.AccountID,
		WorkerID:     item.WorkerID,
		Attempt:      item.Attempt,
		Result:       item.Result,
		ErrorCode:    item.ErrorCode,
		ErrorMessage: item.ErrorMessage,
		StartedAt:    item.StartedAt,
		EndedAt:      item.EndedAt,
	}
}

func (s *octoModuleService) GetVenvInfo(ctx context.Context, typeKey string) (*dto.VenvInfoResponse, error) {
	_ = ctx
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	if info.ModuleDir == "" {
		return nil, invalidInput("module directory could not be resolved")
	}
	venvDir := octomodule.VenvDir(info.ModuleDir)
	pythonPath := octomodule.VenvPythonPath(info.ModuleDir)
	exists := octomodule.VenvExists(info.ModuleDir)
	reqPath := filepath.Join(info.ModuleDir, "requirements.txt")
	hasReq := octomodule.FileExists(reqPath)
	var reqContent string
	if hasReq {
		if data, readErr := os.ReadFile(reqPath); readErr == nil {
			reqContent = string(data)
		}
	}
	return &dto.VenvInfoResponse{
		Exists:              exists,
		Dir:                 venvDir,
		PythonPath:          pythonPath,
		HasRequirements:     hasReq,
		RequirementsContent: reqContent,
	}, nil
}

func (s *octoModuleService) InstallDeps(ctx context.Context, typeKey string, req *dto.InstallDepsRequest) (*dto.InstallDepsResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	info, err := s.Get(ctx, typeKey)
	if err != nil {
		return nil, err
	}
	if info.ModuleDir == "" {
		return nil, invalidInput("module directory could not be resolved")
	}

	var out strings.Builder

	// Create venv if it doesn't exist yet.
	if !octomodule.VenvExists(info.ModuleDir) {
		venvOut, venvErr := runSubprocess(ctx, info.ModuleDir, s.pythonBin, []string{"-m", "venv", ".venv"}, 2*time.Minute)
		out.WriteString(venvOut)
		if venvErr != nil {
			// On Debian/Ubuntu, python3-venv may not be installed; fall back to
			// --without-pip and bootstrap pip separately via ensurepip.
			out.WriteString("\n[fallback] retrying with --without-pip and ensurepip...\n")
			venvOut2, venvErr2 := runSubprocess(ctx, info.ModuleDir, s.pythonBin, []string{"-m", "venv", "--without-pip", ".venv"}, 2*time.Minute)
			out.WriteString(venvOut2)
			if venvErr2 != nil {
				return &dto.InstallDepsResponse{Success: false, Output: out.String()}, nil
			}
			venvPython := octomodule.VenvPythonPath(info.ModuleDir)
			pipOut, pipErr := runSubprocess(ctx, info.ModuleDir, venvPython, []string{"-m", "ensurepip", "--upgrade"}, 2*time.Minute)
			out.WriteString(pipOut)
			if pipErr != nil {
				return &dto.InstallDepsResponse{Success: false, Output: out.String()}, nil
			}
		}
	}

	pipPath := octomodule.VenvPipPath(info.ModuleDir)

	// Write requirements.txt if caller supplied content.
	if req.RequirementsContent != "" {
		reqPath := filepath.Join(info.ModuleDir, "requirements.txt")
		if writeErr := os.WriteFile(reqPath, []byte(req.RequirementsContent), 0644); writeErr != nil {
			return nil, internalError("failed to write requirements.txt", writeErr)
		}
	}

	// Install from requirements.txt.
	if req.FromRequirements || req.RequirementsContent != "" {
		reqPath := filepath.Join(info.ModuleDir, "requirements.txt")
		pipOut, pipErr := runSubprocess(ctx, info.ModuleDir, pipPath, []string{"install", "-r", reqPath}, 5*time.Minute)
		out.WriteString(pipOut)
		if pipErr != nil {
			return &dto.InstallDepsResponse{Success: false, Output: out.String()}, nil
		}
	}

	// Install individual packages.
	if len(req.Packages) > 0 {
		args := append([]string{"install"}, req.Packages...)
		pipOut, pipErr := runSubprocess(ctx, info.ModuleDir, pipPath, args, 5*time.Minute)
		out.WriteString(pipOut)
		if pipErr != nil {
			return &dto.InstallDepsResponse{Success: false, Output: out.String()}, nil
		}
	}

	return &dto.InstallDepsResponse{Success: true, Output: out.String()}, nil
}

func runSubprocess(ctx context.Context, dir, binary string, args []string, timeout time.Duration) (string, error) {
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(execCtx, binary, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

var _ OctoModuleService = (*octoModuleService)(nil)
