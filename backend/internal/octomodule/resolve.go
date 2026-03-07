package octomodule

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResolveResult struct {
	EntryPath string
	Source    string
}

func ResolveEntryPath(moduleDir string, typeKey string, scriptConfig json.RawMessage) (ResolveResult, error) {
	baseDir := strings.TrimSpace(moduleDir)
	if baseDir == "" {
		return ResolveResult{}, fmt.Errorf("octo module directory is empty")
	}
	if strings.TrimSpace(typeKey) == "" {
		return ResolveResult{}, fmt.Errorf("type key is required")
	}

	entry, source, err := parseEntry(scriptConfig)
	if err != nil {
		return ResolveResult{}, err
	}
	if strings.TrimSpace(entry) == "" {
		entry = strings.TrimSpace(typeKey) + "/main.py"
		source = "default"
	}

	resolved, err := resolveEntryWithinBase(baseDir, entry)
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{
		EntryPath: resolved,
		Source:    source,
	}, nil
}

func resolveEntryWithinBase(baseDir string, entry string) (string, error) {
	resolved := strings.TrimSpace(entry)
	if resolved == "" {
		return "", fmt.Errorf("module script entry is empty")
	}
	if filepath.IsAbs(resolved) {
		return "", fmt.Errorf("module script entry must be a relative path under OCTO_MODULE_DIR")
	}

	cleanEntry := filepath.Clean(resolved)
	if cleanEntry == "." || cleanEntry == "" {
		return "", fmt.Errorf("module script entry is invalid")
	}
	parentPrefix := ".." + string(filepath.Separator)
	if cleanEntry == ".." || strings.HasPrefix(cleanEntry, parentPrefix) {
		return "", fmt.Errorf("module script entry cannot escape OCTO_MODULE_DIR")
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve OCTO_MODULE_DIR: %w", err)
	}
	absResolved, err := filepath.Abs(filepath.Join(absBase, cleanEntry))
	if err != nil {
		return "", fmt.Errorf("resolve module script path: %w", err)
	}
	rel, err := filepath.Rel(absBase, absResolved)
	if err != nil {
		return "", fmt.Errorf("validate module script path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, parentPrefix) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("module script entry cannot escape OCTO_MODULE_DIR")
	}

	return filepath.Clean(absResolved), nil
}

// ResolveFileInModuleDir resolves a filename relative to moduleDir with path-traversal protection.
func ResolveFileInModuleDir(moduleDir, filename string) (string, error) {
	base := strings.TrimSpace(moduleDir)
	if base == "" {
		return "", fmt.Errorf("module directory is empty")
	}
	name := strings.TrimPrefix(strings.TrimSpace(filename), "/")
	if name == "" {
		return "", fmt.Errorf("filename is required")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("filename must be a relative path within the module directory")
	}
	cleaned := filepath.Clean(name)
	parentPrefix := ".." + string(filepath.Separator)
	if cleaned == ".." || strings.HasPrefix(cleaned, parentPrefix) {
		return "", fmt.Errorf("filename cannot escape the module directory")
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve module directory: %w", err)
	}
	absPath := filepath.Join(absBase, cleaned)
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, parentPrefix) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("filename cannot escape the module directory")
	}
	return absPath, nil
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func parseEntry(scriptConfig json.RawMessage) (string, string, error) {
	if len(scriptConfig) == 0 {
		return "", "default", nil
	}

	var payload map[string]any
	if err := json.Unmarshal(scriptConfig, &payload); err != nil {
		return "", "", fmt.Errorf("script_config must be a valid JSON object")
	}

	if entry, ok := readEntry(payload); ok {
		return entry, "script_config", nil
	}

	if rawModule, ok := payload["octoModule"].(map[string]any); ok {
		if entry, ok := readEntry(rawModule); ok {
			return entry, "script_config.octoModule", nil
		}
	}

	return "", "default", nil
}

func readEntry(payload map[string]any) (string, bool) {
	keys := []string{"entry", "module_entry", "module_script", "script", "path", "file"}
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		text, ok := raw.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		return trimmed, true
	}
	return "", false
}
