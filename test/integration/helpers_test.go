//go:build integration

// Package integration exercises serv's core subsystems end-to-end using the
// test/fixtures/helper binary as a real, controllable child process. Run
// with: go test -tags=integration ./test/integration/...
package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

var (
	buildHelperOnce sync.Once
	helperPath      string
	buildHelperErr  error
	helperDir       string
)

// TestMain builds the helper fixture into a directory that outlives any
// single test's t.TempDir() (which is removed when that test ends), and
// cleans it up once the whole test binary finishes.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "serv-integration-helper")
	if err == nil {
		helperDir = dir
	}
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}

// helperBinary builds test/fixtures/helper once per test run and returns
// its path.
func helperBinary(t *testing.T) string {
	t.Helper()

	buildHelperOnce.Do(func() {
		if helperDir == "" {
			buildHelperErr = fmt.Errorf("helper build directory was not initialized")
			return
		}

		name := "helper"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		out := filepath.Join(helperDir, name)

		cmd := exec.Command("go", "build", "-o", out, "./../fixtures/helper")
		if output, err := cmd.CombinedOutput(); err != nil {
			buildHelperErr = err
			t.Logf("building helper fixture failed: %v\n%s", err, output)
			return
		}
		helperPath = out
	})

	if buildHelperErr != nil {
		t.Fatalf("helper fixture build failed: %v", buildHelperErr)
	}
	return helperPath
}

// uniqueServiceName returns a service name for prefix that won't collide
// across test runs, so tests remain idempotent even if a previous run's
// cleanup didn't fully complete.
func uniqueServiceName(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("servtest-%s-%d", prefix, time.Now().UnixNano())
}

// baseConfig returns a minimal ServiceConfig pointing at the helper
// fixture, with fast timeouts suitable for tests.
func baseConfig(name, exe string, args ...string) *api.ServiceConfig {
	trueVal := true
	return &api.ServiceConfig{
		Name:       name,
		Executable: exe,
		Arguments:  args,
		Restart: api.RestartConfig{
			Enabled: &trueVal,
		},
		KillProcessTree: &trueVal,
	}
}
