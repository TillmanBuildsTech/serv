package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/internal/platform"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// saveTestConfig writes cfg as YAML to the default config path for name, so
// commands that read the on-disk config (status, config update) can find
// it.
func saveTestConfig(t *testing.T, name string, cfg *api.ServiceConfig) {
	t.Helper()
	path := config.DefaultConfigPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// newTestRootCmd builds a fresh root command with all cli subcommands
// registered, so each test gets isolated flag state.
func newTestRootCmd() *cobra.Command {
	root := &cobra.Command{Use: "serv"}
	Register(root)
	return root
}

func withMockManager(t *testing.T, m *platform.MockManager) {
	t.Helper()
	orig := managerFactory
	managerFactory = func() platform.ServiceManager { return m }
	t.Cleanup(func() { managerFactory = orig })
}

func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newTestRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestInstallWithOnlyExeFlag(t *testing.T) {
	var installed *api.ServiceConfig
	withMockManager(t, &platform.MockManager{
		InstallFunc: func(cfg *api.ServiceConfig) error {
			installed = cfg
			return nil
		},
	})

	out, err := runCmd(t, "install", "--exe", os.Args[0])
	if err != nil {
		t.Fatalf("install: unexpected error: %v (output: %s)", err, out)
	}
	if installed == nil {
		t.Fatal("Install was never called")
	}
	if installed.Executable != os.Args[0] {
		t.Errorf("Executable = %q, want %q", installed.Executable, os.Args[0])
	}
	if installed.Name == "" {
		t.Error("Name should default to the executable's base name, got empty")
	}
	if !strings.Contains(out, "installed") {
		t.Errorf("output = %q, want confirmation message", out)
	}
}

func TestInstallWithConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := dir + "/svc.yaml"
	yaml := "name: myapp\nexecutable: " + os.Args[0] + "\n"
	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	var installed *api.ServiceConfig
	withMockManager(t, &platform.MockManager{
		InstallFunc: func(cfg *api.ServiceConfig) error {
			installed = cfg
			return nil
		},
	})

	if _, err := runCmd(t, "install", "--config", configPath); err != nil {
		t.Fatalf("install: unexpected error: %v", err)
	}
	if installed.Name != "myapp" {
		t.Errorf("Name = %q, want myapp (from config file)", installed.Name)
	}
}

func TestInstallFlagsOverrideConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := dir + "/svc.yaml"
	yaml := "name: fromfile\nexecutable: " + os.Args[0] + "\n"
	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	var installed *api.ServiceConfig
	withMockManager(t, &platform.MockManager{
		InstallFunc: func(cfg *api.ServiceConfig) error {
			installed = cfg
			return nil
		},
	})

	if _, err := runCmd(t, "install", "--config", configPath, "--name", "fromflag"); err != nil {
		t.Fatalf("install: unexpected error: %v", err)
	}
	if installed.Name != "fromflag" {
		t.Errorf("Name = %q, want fromflag (CLI flag should override config file)", installed.Name)
	}
}

func TestInstallValidationFailure(t *testing.T) {
	withMockManager(t, &platform.MockManager{})

	_, err := runCmd(t, "install", "--exe", "/does/not/exist")
	if err == nil {
		t.Fatal("install: expected validation error for nonexistent executable")
	}
}

func TestInstallPropagatesManagerError(t *testing.T) {
	wantErr := errors.New("already exists")
	withMockManager(t, &platform.MockManager{
		InstallFunc: func(cfg *api.ServiceConfig) error { return wantErr },
	})

	_, err := runCmd(t, "install", "--exe", os.Args[0])
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("install: expected wrapped manager error, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	var removed string
	withMockManager(t, &platform.MockManager{
		RemoveFunc: func(name string) error { removed = name; return nil },
	})

	out, err := runCmd(t, "remove", "myapp")
	if err != nil {
		t.Fatalf("remove: unexpected error: %v", err)
	}
	if removed != "myapp" {
		t.Errorf("Remove called with %q, want myapp", removed)
	}
	if !strings.Contains(out, "removed") {
		t.Errorf("output = %q, want confirmation message", out)
	}
}

func TestStartStopRestart(t *testing.T) {
	var started, stopped, restarted string
	withMockManager(t, &platform.MockManager{
		StartFunc:   func(name string) error { started = name; return nil },
		StopFunc:    func(name string) error { stopped = name; return nil },
		RestartFunc: func(name string) error { restarted = name; return nil },
	})

	if _, err := runCmd(t, "start", "myapp"); err != nil {
		t.Fatalf("start: unexpected error: %v", err)
	}
	if _, err := runCmd(t, "stop", "myapp"); err != nil {
		t.Fatalf("stop: unexpected error: %v", err)
	}
	if _, err := runCmd(t, "restart", "myapp"); err != nil {
		t.Fatalf("restart: unexpected error: %v", err)
	}

	if started != "myapp" || stopped != "myapp" || restarted != "myapp" {
		t.Errorf("start/stop/restart = %q/%q/%q, want myapp/myapp/myapp", started, stopped, restarted)
	}
}

func TestStatus(t *testing.T) {
	withMockManager(t, &platform.MockManager{
		StatusFunc: func(name string) (platform.ServiceStatus, error) {
			return platform.ServiceStatus{State: "running", PID: 4242}, nil
		},
	})

	origStartTime := processStartTime
	fixedStart := time.Now().Add(-90 * time.Second)
	processStartTime = func(pid int) (time.Time, bool) { return fixedStart, true }
	t.Cleanup(func() { processStartTime = origStartTime })

	out, err := runCmd(t, "status", "myapp")
	if err != nil {
		t.Fatalf("status: unexpected error: %v", err)
	}

	for _, want := range []string{"Name:   myapp", "State:  running", "PID:    4242", "Uptime: 1m30s"} {
		if !strings.Contains(out, want) {
			t.Errorf("status output missing %q; got:\n%s", want, out)
		}
	}
}

func TestStatusNotRunning(t *testing.T) {
	withMockManager(t, &platform.MockManager{
		StatusFunc: func(name string) (platform.ServiceStatus, error) {
			return platform.ServiceStatus{State: "stopped", PID: 0}, nil
		},
	})

	out, err := runCmd(t, "status", "myapp")
	if err != nil {
		t.Fatalf("status: unexpected error: %v", err)
	}
	if !strings.Contains(out, "PID:    -") || !strings.Contains(out, "Uptime: -") {
		t.Errorf("expected '-' placeholders for stopped service; got:\n%s", out)
	}
}

func TestList(t *testing.T) {
	withMockManager(t, &platform.MockManager{
		ListFunc: func() ([]platform.ServiceInfo, error) {
			return []platform.ServiceInfo{
				{Name: "svc1", State: "running", PID: 100, DisplayName: "Service One"},
				{Name: "svc2", State: "stopped", PID: 0, DisplayName: "Service Two"},
			}, nil
		},
	})

	out, err := runCmd(t, "list")
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if !strings.Contains(out, "svc1") || !strings.Contains(out, "running") || !strings.Contains(out, "100") {
		t.Errorf("list output missing svc1 details; got:\n%s", out)
	}
	if !strings.Contains(out, "svc2") || !strings.Contains(out, "stopped") {
		t.Errorf("list output missing svc2 details; got:\n%s", out)
	}
}

func TestListEmpty(t *testing.T) {
	withMockManager(t, &platform.MockManager{
		ListFunc: func() ([]platform.ServiceInfo, error) { return nil, nil },
	})

	out, err := runCmd(t, "list")
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if !strings.Contains(out, "No services installed") {
		t.Errorf("output = %q, want empty-list message", out)
	}
}

func TestConfigUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROGRAMDATA", dir)

	existing := &api.ServiceConfig{Name: "myapp", Executable: os.Args[0]}
	saveTestConfig(t, "myapp", existing)

	var updated *api.ServiceConfig
	withMockManager(t, &platform.MockManager{
		UpdateConfigFunc: func(name string, cfg *api.ServiceConfig) error {
			updated = cfg
			return nil
		},
	})

	if _, err := runCmd(t, "config", "myapp", "--description", "new description"); err != nil {
		t.Fatalf("config: unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateConfig was never called")
	}
	if updated.Description != "new description" {
		t.Errorf("Description = %q, want %q", updated.Description, "new description")
	}
	if updated.Executable != os.Args[0] {
		t.Errorf("Executable = %q, want unchanged %q", updated.Executable, os.Args[0])
	}
}
