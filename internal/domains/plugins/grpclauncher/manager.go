package grpclauncher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const logScannerMaxTokenSize = 1024 * 1024

type ProcessConfig struct {
	Key        string
	Address    string
	Directory  string
	Entrypoint string
}

type Manager struct {
	logger         *zap.Logger
	pythonBin      string
	sdkDir         string
	restartBackoff time.Duration
	processes      []ProcessConfig
}

func New(logger *zap.Logger, pythonBin, sdkDir string, processes []ProcessConfig) *Manager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Manager{
		logger:         logger,
		pythonBin:      strings.TrimSpace(pythonBin),
		sdkDir:         strings.TrimSpace(sdkDir),
		restartBackoff: time.Second,
		processes:      append([]ProcessConfig(nil), processes...),
	}
}

func Discover(modulesDir string, services map[string]string) []ProcessConfig {
	keys := make([]string, 0, len(services))
	for key, address := range services {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(address) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	processes := make([]ProcessConfig, 0, len(keys))
	for _, key := range keys {
		directory := resolvePluginDirectory(modulesDir, key)
		if directory == "" {
			continue
		}
		if abs, err := filepath.Abs(directory); err == nil {
			directory = abs
		}

		entrypoint := filepath.Join(directory, "main.py")
		if _, err := os.Stat(entrypoint); err != nil {
			continue
		}

		processes = append(processes, ProcessConfig{
			Key:        strings.TrimSpace(key),
			Address:    strings.TrimSpace(services[key]),
			Directory:  directory,
			Entrypoint: "main.py",
		})
	}
	return processes
}

func (m *Manager) Run(ctx context.Context) error {
	if len(m.processes) == 0 {
		m.logger.Info("no local plugin gRPC services configured for worker launch")
		<-ctx.Done()
		return nil
	}

	var wg sync.WaitGroup
	for _, process := range m.processes {
		process := process
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.runProcess(ctx, process)
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

func (m *Manager) runProcess(ctx context.Context, process ProcessConfig) {
	logger := m.logger.With(
		zap.String("plugin_key", process.Key),
		zap.String("address", process.Address),
	)

	pythonExec := ""
	for ctx.Err() == nil {
		if pythonExec == "" {
			preparedPython, err := m.prepareVirtualEnv(ctx, process, logger)
			if err != nil {
				logger.Error("prepare plugin virtualenv failed", zap.Error(err))
				if !sleepWithContext(ctx, m.restartBackoff) {
					return
				}
				continue
			}
			pythonExec = preparedPython
		}

		cmd := exec.CommandContext(
			ctx,
			pythonExec,
			process.Entrypoint,
			"--grpc",
			"--address",
			process.Address,
		)
		cmd.Dir = process.Directory
		cmd.Env = append(os.Environ(), "PYTHONPATH="+buildPythonPath(m.sdkDir))

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.Error("open plugin stdout failed", zap.Error(err))
			if !sleepWithContext(ctx, m.restartBackoff) {
				return
			}
			continue
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			logger.Error("open plugin stderr failed", zap.Error(err))
			if !sleepWithContext(ctx, m.restartBackoff) {
				return
			}
			continue
		}

		if err := cmd.Start(); err != nil {
			logger.Error("start plugin process failed", zap.Error(err))
			if !sleepWithContext(ctx, m.restartBackoff) {
				return
			}
			continue
		}

		logger.Info("plugin process started", zap.Int("pid", cmd.Process.Pid))

		var streamWG sync.WaitGroup
		streamWG.Add(2)
		go func() {
			defer streamWG.Done()
			streamProcessLogs(ctx, logger, stdout, false)
		}()
		go func() {
			defer streamWG.Done()
			streamProcessLogs(ctx, logger, stderr, true)
		}()

		err = cmd.Wait()
		streamWG.Wait()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			logger.Warn("plugin process exited unexpectedly", zap.Error(err))
		} else {
			logger.Warn("plugin process exited; restarting")
		}
		if !sleepWithContext(ctx, m.restartBackoff) {
			return
		}
	}
}

func (m *Manager) prepareVirtualEnv(ctx context.Context, process ProcessConfig, logger *zap.Logger) (string, error) {
	venvDir := filepath.Join(process.Directory, ".venv")
	venvPython := virtualEnvPythonPath(venvDir)
	if _, err := os.Stat(venvPython); err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat plugin virtualenv python %s: %w", venvPython, err)
		}
		logger.Info("creating plugin virtualenv", zap.String("venv_dir", venvDir))
		if err := m.runSetupCommand(ctx, logger, process.Directory, m.pythonBinary(), "-m", "venv", venvDir); err != nil {
			return "", err
		}
	}

	requirementsPath := filepath.Join(process.Directory, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		logger.Info("installing plugin dependencies", zap.String("requirements", requirementsPath))
		if err := m.runSetupCommand(
			ctx,
			logger,
			process.Directory,
			venvPython,
			"-m",
			"pip",
			"install",
			"--disable-pip-version-check",
			"-r",
			requirementsPath,
		); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat requirements %s: %w", requirementsPath, err)
	}

	return venvPython, nil
}

func (m *Manager) runSetupCommand(ctx context.Context, logger *zap.Logger, workdir, bin string, args ...string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workdir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open setup stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open setup stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start setup command %s %s: %w", bin, strings.Join(args, " "), err)
	}

	var streamWG sync.WaitGroup
	streamWG.Add(2)
	go func() {
		defer streamWG.Done()
		streamProcessLogs(ctx, logger, stdout, false)
	}()
	go func() {
		defer streamWG.Done()
		streamProcessLogs(ctx, logger, stderr, true)
	}()

	err = cmd.Wait()
	streamWG.Wait()
	if err != nil {
		return fmt.Errorf("run setup command %s %s: %w", bin, strings.Join(args, " "), err)
	}
	return nil
}

func streamProcessLogs(ctx context.Context, logger *zap.Logger, reader io.Reader, isErr bool) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), logScannerMaxTokenSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if isErr {
			logger.Warn("plugin stderr", zap.String("line", line))
			continue
		}
		logger.Info("plugin stdout", zap.String("line", line))
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		logger.Warn("read plugin process output failed", zap.Error(err))
	}
}

func resolvePluginDirectory(modulesDir, key string) string {
	modulesDir = strings.TrimSpace(modulesDir)
	key = strings.TrimSpace(key)
	if modulesDir == "" || key == "" {
		return ""
	}

	candidates := []string{
		key,
		strings.ReplaceAll(key, "-", "_"),
		strings.ReplaceAll(key, "_", "-"),
	}
	for _, candidate := range candidates {
		directory := filepath.Join(modulesDir, candidate)
		if info, err := os.Stat(directory); err == nil && info.IsDir() {
			return directory
		}
	}
	return ""
}

func buildPythonPath(sdkDir string) string {
	parts := make([]string, 0, 2)
	if trimmed := strings.TrimSpace(sdkDir); trimmed != "" {
		if abs, err := filepath.Abs(trimmed); err == nil {
			parts = append(parts, abs)
		} else {
			parts = append(parts, trimmed)
		}
	}
	if existing := strings.TrimSpace(os.Getenv("PYTHONPATH")); existing != "" {
		parts = append(parts, existing)
	}
	return strings.Join(parts, string(os.PathListSeparator))
}

func virtualEnvPythonPath(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

func (m *Manager) pythonBinary() string {
	if strings.TrimSpace(m.pythonBin) != "" {
		return strings.TrimSpace(m.pythonBin)
	}
	return "python3"
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
