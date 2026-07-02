package io

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func newTestRotatingWriter(t *testing.T, cfg api.LogRotationConfig) (*RotatingWriter, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := NewRotatingWriter(path, cfg)
	if err != nil {
		t.Fatalf("NewRotatingWriter: %v", err)
	}
	t.Cleanup(func() { w.Close() })
	return w, path
}

func listRotatedFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.Name() != "app.log" {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestRotatingWriterSizeBasedRotation(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{
		Enabled:  true,
		MaxBytes: 20,
	})

	line := []byte("0123456789\n") // 11 bytes
	// Write 1: size 0+11=11 <= 20, no rotation. size becomes 11.
	if _, err := w.Write(line); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Write 2: size 11+11=22 > 20, rotates before writing. size becomes 11.
	if _, err := w.Write(line); err != nil {
		t.Fatalf("Write: %v", err)
	}

	rotated := listRotatedFiles(t, filepath.Dir(path))
	if len(rotated) != 1 {
		t.Fatalf("expected 1 rotated file, got %v", rotated)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(line) {
		t.Fatalf("current log file = %q, want only the post-rotation line %q", data, line)
	}
}

// TestRotatingWriterCollidingRotationsGetUniqueNames verifies that two
// rotations happening within the same second (e.g. under a MinInterval of
// 0) don't silently overwrite each other's rotated file.
func TestRotatingWriterCollidingRotationsGetUniqueNames(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{
		Enabled:  true,
		MaxBytes: 5,
	})

	fixed := time.Date(2026, 6, 19, 14, 30, 22, 0, time.UTC)
	w.now = func() time.Time { return fixed }

	for i := 0; i < 3; i++ {
		if _, err := w.Write([]byte("0123456789\n")); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	// Every 11-byte line exceeds the 5-byte cap on its own, so each of the
	// 3 writes triggers a rotation before it lands, producing 3 distinct
	// rotated files despite the fixed (colliding) rotation timestamp.
	rotated := listRotatedFiles(t, filepath.Dir(path))
	if len(rotated) != 3 {
		t.Fatalf("expected 3 uniquely-named rotated files, got %v", rotated)
	}
}

func TestRotatingWriterRotatesAtLineBoundary(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{
		Enabled:  true,
		MaxBytes: 5,
	})

	lines := [][]byte{
		[]byte("aaaaaaaaaa\n"),
		[]byte("bbbbbbbbbb\n"),
		[]byte("cccccccccc\n"),
	}
	for _, l := range lines {
		if _, err := w.Write(l); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Every write is a complete line, so no rotated or current file should
	// ever contain a partial line.
	if bytes.Count(data, []byte("\n")) != bytes.Count(bytes.TrimRight(data, "\n"), []byte("\n"))+1 && len(data) > 0 {
		t.Fatalf("current file does not end at a line boundary: %q", data)
	}

	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			t.Fatalf("file %s does not end at a line boundary: %q", e.Name(), content)
		}
	}
}

func TestRotatingWriterTimeBasedRotation(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{
		Enabled: true,
		MaxAge:  api.Duration(10 * time.Millisecond),
	})

	now := time.Now()
	w.now = func() time.Time { return now }

	if _, err := w.Write([]byte("first\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if rotated := listRotatedFiles(t, filepath.Dir(path)); len(rotated) != 0 {
		t.Fatalf("unexpected rotation before max age elapsed: %v", rotated)
	}

	now = now.Add(11 * time.Millisecond)
	if _, err := w.Write([]byte("second\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	rotated := listRotatedFiles(t, filepath.Dir(path))
	if len(rotated) != 1 {
		t.Fatalf("expected 1 rotated file after max age elapsed, got %v", rotated)
	}
}

func TestRotatingWriterMinIntervalPreventsRapidRotation(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{
		Enabled:     true,
		MaxBytes:    1, // rotate on almost every write
		MinInterval: api.Duration(time.Hour),
	})

	now := time.Now()
	w.now = func() time.Time { return now }

	for i := 0; i < 5; i++ {
		if _, err := w.Write([]byte("line\n")); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	// The very first rotation (size-triggered) is allowed since lastRotation
	// starts zero; subsequent writes stay within the 1-hour min interval, so
	// no further rotations should occur.
	rotated := listRotatedFiles(t, filepath.Dir(path))
	if len(rotated) != 1 {
		t.Fatalf("expected exactly 1 rotation due to min interval, got %v", rotated)
	}
}

func TestRotatingWriterExplicitRotate(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{Enabled: true})

	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Rotate(); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	rotated := listRotatedFiles(t, filepath.Dir(path))
	if len(rotated) != 1 {
		t.Fatalf("expected 1 rotated file, got %v", rotated)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected fresh log file to exist at original path: %v", err)
	}
}

func TestRotatingWriterTimestampLines(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{TimestampLines: true})

	fixed := time.Date(2026, 6, 19, 14, 30, 22, 123000000, time.UTC)
	w.now = func() time.Time { return fixed }

	if _, err := w.Write([]byte("hello world\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "[2026-06-19 14:30:22.123] hello world\n"
	if string(data) != want {
		t.Fatalf("data = %q, want %q", data, want)
	}
}

func TestRotatingWriterNoTimestampByDefault(t *testing.T) {
	w, path := newTestRotatingWriter(t, api.LogRotationConfig{})

	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("data = %q, want %q", data, "hello\n")
	}
}

func TestRotatedFilePathFormat(t *testing.T) {
	at := time.Date(2026, 6, 19, 14, 30, 22, 0, time.UTC)
	got := rotatedFilePath(filepath.Join("var", "log", "app.log"), at)
	want := filepath.Join("var", "log", "app-20260619T143022.log")
	if got != want {
		t.Fatalf("rotatedFilePath = %q, want %q", got, want)
	}
}
