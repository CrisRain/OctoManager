package grpclauncher

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverFindsConfiguredPluginsByDirectoryName(t *testing.T) {
	root := t.TempDir()
	underscoreDir := filepath.Join(root, "octo_demo")
	if err := os.MkdirAll(underscoreDir, 0o755); err != nil {
		t.Fatalf("mkdir underscore dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(underscoreDir, "main.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("write main.py: %v", err)
	}

	hyphenDir := filepath.Join(root, "hello-world")
	if err := os.MkdirAll(hyphenDir, 0o755); err != nil {
		t.Fatalf("mkdir hyphen dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hyphenDir, "main.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("write hyphen main.py: %v", err)
	}

	processes := Discover(root, map[string]string{
		"octo_demo":   "127.0.0.1:50051",
		"hello_world": "127.0.0.1:50052",
		"missing":     "127.0.0.1:50053",
	})

	if len(processes) != 2 {
		t.Fatalf("expected 2 processes, got %d", len(processes))
	}
	if processes[0].Key != "hello_world" || processes[0].Directory != hyphenDir {
		t.Fatalf("unexpected first process %#v", processes[0])
	}
	if processes[1].Key != "octo_demo" || processes[1].Directory != underscoreDir {
		t.Fatalf("unexpected second process %#v", processes[1])
	}
}

func TestBuildPythonPathPreservesExistingPath(t *testing.T) {
	t.Setenv("PYTHONPATH", "/tmp/existing")
	got := buildPythonPath("plugins/sdk/python")
	if got == "" {
		t.Fatalf("expected python path")
	}
	if filepath.Base(got) == got {
		t.Fatalf("expected sdk dir to be expanded into a path, got %q", got)
	}
	if got[len(got)-len("/tmp/existing"):] != "/tmp/existing" {
		t.Fatalf("expected existing PYTHONPATH suffix, got %q", got)
	}
}

func TestVirtualEnvPythonPath(t *testing.T) {
	got := virtualEnvPythonPath(filepath.Join("plugins", "modules", "octo_demo", ".venv"))
	if runtime.GOOS == "windows" {
		want := filepath.Join("plugins", "modules", "octo_demo", ".venv", "Scripts", "python.exe")
		if got != want {
			t.Fatalf("unexpected windows venv python path %q", got)
		}
		return
	}

	want := filepath.Join("plugins", "modules", "octo_demo", ".venv", "bin", "python")
	if got != want {
		t.Fatalf("unexpected venv python path %q", got)
	}
}
