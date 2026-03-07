package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type Input struct {
	Action  string         `json:"action"`
	Account InputAccount   `json:"account"`
	Params  map[string]any `json:"params"`
	Context InputContext   `json:"context"`
}

type InputAccount struct {
	Identifier string         `json:"identifier"`
	Spec       map[string]any `json:"spec"`
}

type InputContext struct {
	TenantID  string `json:"tenant_id"`
	RequestID string `json:"request_id"`
}

type Output struct {
	Status       string         `json:"status"`
	Result       map[string]any `json:"result,omitempty"`
	Session      *OutputSession `json:"session,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type OutputSession struct {
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	ExpiresAt string         `json:"expires_at,omitempty"`
}

type PythonBridge struct {
	Binary  string
	Script  string
	Timeout time.Duration
}

func resolveVenvPython(dir string) string {
	var candidate string
	if runtime.GOOS == "windows" {
		candidate = filepath.Join(dir, ".venv", "Scripts", "python.exe")
	} else {
		candidate = filepath.Join(dir, ".venv", "bin", "python")
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func (p PythonBridge) Execute(ctx context.Context, input Input) (Output, error) {
	return p.ExecuteWithScript(ctx, p.Script, input)
}

func (p PythonBridge) ExecuteWithScript(ctx context.Context, scriptPath string, input Input) (Output, error) {
	rawInput, err := json.Marshal(input)
	if err != nil {
		return Output{}, err
	}

	if scriptPath == "" {
		return Output{}, fmt.Errorf("python script path is required")
	}

	execCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	binary := p.Binary
	if venvPython := resolveVenvPython(filepath.Dir(scriptPath)); venvPython != "" {
		binary = venvPython
	}
	cmd := exec.CommandContext(execCtx, binary, scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	cmd.Env = []string{
		"PYTHONUNBUFFERED=1",
		"PYTHONIOENCODING=UTF-8",
	}
	cmd.Stdin = bytes.NewReader(rawInput)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return Output{}, fmt.Errorf("python script timed out after %s", p.Timeout)
		}
		return Output{}, fmt.Errorf("python script failed: %w (stderr=%s)", err, stderr.String())
	}

	var output Output
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return Output{}, fmt.Errorf("invalid python output: %w (stdout=%s, stderr=%s)", err, stdout.String(), stderr.String())
	}
	return output, nil
}
