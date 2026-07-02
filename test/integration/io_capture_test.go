//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ioredirect "github.com/TillmanBuildsTech/serv/internal/io"
	"github.com/TillmanBuildsTech/serv/internal/process"
)

// TestStdoutStderrCapture launches the helper fixture with stdout capture
// configured and confirms its periodic output lands in the log file.
func TestStdoutStderrCapture(t *testing.T) {
	exe := helperBinary(t)
	logPath := filepath.Join(t.TempDir(), "out.log")

	cfg := baseConfig("iocapture", exe, "-output-interval=20ms", "-exit-after=300ms")
	cfg.Stdout = logPath

	var redirect *ioredirect.Redirect
	mp, err := process.StartProcess(cfg, func(cmd *exec.Cmd) error {
		r, err := ioredirect.Setup(cmd, cfg, ioredirect.Options{})
		redirect = r
		return err
	})
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}

	select {
	case <-mp.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("helper did not exit within timeout")
	}

	// Close after the child has exited, once os/exec's internal pipe-copy
	// goroutines have finished draining stdout, so the flushed content is
	// fully visible and the log file handle is released before the test's
	// TempDir cleanup runs.
	if err := redirect.Close(); err != nil {
		t.Fatalf("Redirect.Close: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(data), "tick 1") {
		t.Errorf("log file missing expected output; got:\n%s", data)
	}
}
