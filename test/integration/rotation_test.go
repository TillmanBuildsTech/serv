//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	ioredirect "github.com/TillmanBuildsTech/serv/internal/io"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// TestLogRotationBySize writes enough lines through a RotatingWriter to
// cross its size threshold and confirms a rotated, timestamped file
// appears alongside the active log.
func TestLogRotationBySize(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	w, err := ioredirect.NewRotatingWriter(logPath, api.LogRotationConfig{
		Enabled:  true,
		MaxBytes: 200,
	})
	if err != nil {
		t.Fatalf("NewRotatingWriter: %v", err)
	}
	defer w.Close()

	line := []byte("this is a log line that repeats to exceed the size threshold\n")
	for i := 0; i < 10; i++ {
		if _, err := w.Write(line); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var rotatedCount int
	for _, e := range entries {
		if e.Name() != "app.log" {
			rotatedCount++
		}
	}
	if rotatedCount == 0 {
		t.Fatal("expected at least one rotated log file, found none")
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected active log file to still exist: %v", err)
	}
}
