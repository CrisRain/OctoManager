package octomodule

import (
	"os"
	"path/filepath"
	"runtime"
)

func VenvDir(moduleDir string) string {
	return filepath.Join(moduleDir, ".venv")
}

func VenvPythonPath(moduleDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(moduleDir, ".venv", "Scripts", "python.exe")
	}
	return filepath.Join(moduleDir, ".venv", "bin", "python")
}

func VenvPipPath(moduleDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(moduleDir, ".venv", "Scripts", "pip.exe")
	}
	return filepath.Join(moduleDir, ".venv", "bin", "pip")
}

func VenvExists(moduleDir string) bool {
	_, err := os.Stat(VenvPythonPath(moduleDir))
	return err == nil
}
