//go:build windows

package service

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// writeTestConfig writes cfg as YAML to the default config path for name,
// so reload() (which reads from disk) can pick it up.
func writeTestConfig(t *testing.T, name string, cfg *api.ServiceConfig) {
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

// TestMain re-executes the test binary itself as a controllable child
// process when GO_HELPER_MODE is set.
func TestMain(m *testing.M) {
	switch os.Getenv("GO_HELPER_MODE") {
	case "exit_code":
		code, _ := strconv.Atoi(os.Getenv("HELPER_EXIT_CODE"))
		os.Exit(code)
	case "sleep":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func testConfig(mode string, extraEnv map[string]string) *api.ServiceConfig {
	env := map[string]string{"GO_HELPER_MODE": mode}
	for k, v := range extraEnv {
		env[k] = v
	}
	return &api.ServiceConfig{
		Name:        "test-service",
		Executable:  os.Args[0],
		Environment: env,
		Restart: api.RestartConfig{
			Enabled:     api.BoolPtr(true),
			Delay:       api.Duration(time.Millisecond),
			ThrottleCap: api.Duration(50 * time.Millisecond),
		},
		StopMethod: api.StopConfig{
			ConsoleTimeout:   api.Duration(10 * time.Millisecond),
			WindowTimeout:    api.Duration(10 * time.Millisecond),
			ThreadsTimeout:   api.Duration(10 * time.Millisecond),
			TerminateTimeout: api.Duration(200 * time.Millisecond),
		},
	}
}

func TestStartChildSuccess(t *testing.T) {
	cfg := testConfig("exit_code", map[string]string{"HELPER_EXIT_CODE": "0"})
	rt := newRuntimeState("test-service", cfg)

	if err := rt.startChild(); err != nil {
		t.Fatalf("startChild: unexpected error: %v", err)
	}
	if rt.mp == nil || rt.mp.PID <= 0 {
		t.Fatal("startChild: expected a running child with a valid PID")
	}

	<-rt.done() // let the short-lived helper exit before the test returns
}

func TestStartChildAbortedByPreStartHook(t *testing.T) {
	cfg := testConfig("exit_code", map[string]string{"HELPER_EXIT_CODE": "0"})
	cfg.Hooks = map[string]string{"pre-start": "cmd.exe /C exit 1"}
	rt := newRuntimeState("test-service", cfg)

	if err := rt.startChild(); err == nil {
		t.Fatal("startChild: expected error when pre-start hook fails")
	}
	if rt.mp != nil {
		t.Error("startChild: expected no child to be started after hook abort")
	}
}

func TestHandleExitResolvesConfiguredAction(t *testing.T) {
	cases := map[api.ExitAction]exitAction{
		api.ExitActionRestart: actionRestart,
		api.ExitActionIgnore:  actionIgnore,
		api.ExitActionExit:    actionExit,
		api.ExitActionCrash:   actionCrash,
	}

	for cfgAction, want := range cases {
		t.Run(string(cfgAction), func(t *testing.T) {
			cfg := testConfig("exit_code", map[string]string{"HELPER_EXIT_CODE": "3"})
			cfg.ExitActions = map[int]api.ExitAction{3: cfgAction}
			rt := newRuntimeState("test-service", cfg)

			if err := rt.startChild(); err != nil {
				t.Fatalf("startChild: %v", err)
			}
			<-rt.done()

			got := rt.handleExit()
			if got != want {
				t.Errorf("handleExit() = %v, want %v", got, want)
			}
			if want != actionRestart && rt.mp != nil {
				t.Errorf("expected rt.mp to be cleared for action %v", want)
			}
		})
	}
}

func TestNextBackoffEscalates(t *testing.T) {
	cfg := testConfig("exit_code", nil)
	rt := newRuntimeState("test-service", cfg)
	rt.throttle.RecordStart(time.Now())

	first := rt.nextBackoff()
	second := rt.nextBackoff()
	if second <= first {
		t.Errorf("nextBackoff did not escalate: first=%v second=%v", first, second)
	}
}

func TestReloadRestartsChildWithNewConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROGRAMDATA", dir)

	name := "reload-test-service"
	cfg := testConfig("sleep", nil)
	cfg.Name = name

	// Write a config to disk that reload() will pick up: a different exit
	// code helper mode so we can confirm the new config actually took
	// effect on the resulting child.
	newCfg := testConfig("exit_code", map[string]string{"HELPER_EXIT_CODE": "0"})
	newCfg.Name = name
	writeTestConfig(t, name, newCfg)

	rt := newRuntimeState(name, cfg)
	if err := rt.startChild(); err != nil {
		t.Fatalf("startChild: %v", err)
	}
	firstPID := rt.mp.PID

	if err := rt.reload(); err != nil {
		t.Fatalf("reload: unexpected error: %v", err)
	}

	if rt.mp == nil {
		t.Fatal("reload: expected a new child to be running")
	}
	if rt.mp.PID == firstPID {
		t.Error("reload: expected a new child process, got the same PID")
	}
	if rt.cfg.Environment["GO_HELPER_MODE"] != "exit_code" {
		t.Errorf("reload: expected runtime config to be updated, got %+v", rt.cfg.Environment)
	}

	rt.stopChild()
}

func TestStopChildTerminatesRealProcess(t *testing.T) {
	cfg := testConfig("sleep", nil)
	rt := newRuntimeState("test-service", cfg)

	if err := rt.startChild(); err != nil {
		t.Fatalf("startChild: %v", err)
	}

	done := make(chan struct{})
	go func() {
		rt.stopChild()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("stopChild did not return within timeout")
	}

	select {
	case <-rt.done():
	default:
		t.Error("expected child to be marked done after stopChild")
	}
}

func TestStopChildNoActiveChildIsNoop(t *testing.T) {
	cfg := testConfig("exit_code", nil)
	rt := newRuntimeState("test-service", cfg)
	// rt.mp is nil; stopChild must not panic or block.
	rt.stopChild()
}

func TestWiredCallsForRestartCycle(t *testing.T) {
	origResolve := resolveExitActionFn
	origShutdown := shutdownFn
	origKillTree := killTreeFn
	t.Cleanup(func() {
		resolveExitActionFn = origResolve
		shutdownFn = origShutdown
		killTreeFn = origKillTree
	})

	var shutdownCalled, killTreeCalled bool
	shutdownFn = func(ctx context.Context, pid int, done <-chan struct{}, sc api.StopConfig) error {
		shutdownCalled = true
		return nil // deliberately don't kill anything here; killTreeFn below does
	}
	killTreeFn = func(pid int) error {
		killTreeCalled = true
		return origKillTree(pid) // actually terminate so stopChild's <-mp.Done() unblocks
	}

	cfg := testConfig("sleep", nil)
	rt := newRuntimeState("test-service", cfg)
	if err := rt.startChild(); err != nil {
		t.Fatalf("startChild: %v", err)
	}

	rt.stopChild()

	if !shutdownCalled {
		t.Error("expected shutdownFn to be called")
	}
	if !killTreeCalled {
		t.Error("expected killTreeFn to be called")
	}
}
