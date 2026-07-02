package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// TestMain re-executes the test binary itself as a hook process when
// GO_HOOK_MODE is set, allowing tests to run real hook commands without
// depending on external scripts.
func TestMain(m *testing.M) {
	switch os.Getenv("GO_HOOK_MODE") {
	case "dump_env":
		dumpEnv()
		os.Exit(0)
	case "fail":
		os.Exit(1)
	case "sleep":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func dumpEnv() {
	path := os.Getenv("HOOK_OUTPUT_FILE")
	f, err := os.Create(path)
	if err != nil {
		os.Exit(3)
	}
	defer f.Close()

	for _, key := range []string{
		"SERV_SERVICE_NAME", "SERV_PID", "SERV_EXIT_CODE",
		"SERV_RUNTIME_SECONDS", "SERV_EVENT", "SERV_ACTION",
		"SERV_EXE", "SERV_ARGS",
	} {
		fmt.Fprintf(f, "%s=%s\n", key, os.Getenv(key))
	}
}

// quotedSelf returns a command string that runs this test binary itself.
// buildCommand passes it as a single exec.Command argument, which Go
// automatically quotes/escapes as needed on both Windows and Unix, so no
// manual quoting is needed (and would in fact double-escape on Windows).
func quotedSelf() string {
	return os.Args[0]
}

func TestRunNoHookConfigured(t *testing.T) {
	cfg := &api.ServiceConfig{}
	if err := Run(cfg, Context{Event: EventPreStart}, time.Second); err != nil {
		t.Fatalf("Run: expected nil for unconfigured hook, got %v", err)
	}
}

func TestRunExecutesHookWithEnv(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "env.txt")
	t.Setenv("GO_HOOK_MODE", "dump_env")
	t.Setenv("HOOK_OUTPUT_FILE", outputFile)

	cfg := &api.ServiceConfig{
		Hooks: map[string]string{
			"pre-start": quotedSelf(),
		},
	}

	ctx := Context{
		ServiceName:    "myapp",
		PID:            4242,
		ExitCode:       7,
		RuntimeSeconds: 99,
		Event:          EventPreStart,
		Action:         "restart",
		Exe:            "/usr/bin/myapp",
		Args:           []string{"--flag", "value"},
	}

	if err := Run(cfg, ctx, 5*time.Second); err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("reading hook output: %v", err)
	}

	want := map[string]string{
		"SERV_SERVICE_NAME":    "myapp",
		"SERV_PID":             "4242",
		"SERV_EXIT_CODE":       "7",
		"SERV_RUNTIME_SECONDS": "99",
		"SERV_EVENT":           "pre-start",
		"SERV_ACTION":          "restart",
		"SERV_EXE":             "/usr/bin/myapp",
		"SERV_ARGS":            "--flag value",
	}
	got := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok {
			got[k] = v
		}
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %q, want %q", k, got[k], w)
		}
	}
}

func TestRunHookFailureReturnsError(t *testing.T) {
	t.Setenv("GO_HOOK_MODE", "fail")

	cfg := &api.ServiceConfig{
		Hooks: map[string]string{
			"pre-start": quotedSelf(),
		},
	}

	err := Run(cfg, Context{Event: EventPreStart}, 5*time.Second)
	if err == nil {
		t.Fatal("Run: expected error for non-zero hook exit")
	}
}

func TestRunHookTimeout(t *testing.T) {
	t.Setenv("GO_HOOK_MODE", "sleep")

	cfg := &api.ServiceConfig{
		Hooks: map[string]string{
			"pre-stop": quotedSelf(),
		},
	}

	start := time.Now()
	err := Run(cfg, Context{Event: EventPreStop}, 200*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run: expected timeout error")
	}
	if elapsed > 5*time.Second {
		t.Errorf("Run took %v to return after timeout, want well under 5s", elapsed)
	}
}

func TestRunUsesDefaultTimeoutWhenUnset(t *testing.T) {
	if DefaultTimeout != 60*time.Second {
		t.Fatalf("DefaultTimeout = %v, want 60s", DefaultTimeout)
	}
}
