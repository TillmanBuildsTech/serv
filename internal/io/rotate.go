package io

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

const rotationTimestampLayout = "20060102T150405"

// RotatingWriter is an io.WriteCloser that writes to a log file at a fixed
// path, transparently rotating it to a timestamped filename when it grows
// past a configured size or age. Every Write call is expected to carry
// exactly one complete, newline-terminated log line (which is how Logger
// calls it), so rotation only ever happens at a line boundary.
type RotatingWriter struct {
	path string
	cfg  api.LogRotationConfig
	// now is overridable in tests to control the rotation clock.
	now func() time.Time

	mu           sync.Mutex
	file         *os.File
	size         int64
	openedAt     time.Time
	lastRotation time.Time
}

// NewRotatingWriter opens (creating if necessary) the log file at path and
// returns a writer that rotates it according to cfg.
func NewRotatingWriter(path string, cfg api.LogRotationConfig) (*RotatingWriter, error) {
	f, err := openLogFile(path, false)
	if err != nil {
		return nil, err
	}

	var size int64
	if info, err := f.Stat(); err == nil {
		size = info.Size()
	}

	now := time.Now()
	return &RotatingWriter{
		path:     path,
		cfg:      cfg,
		now:      time.Now,
		file:     f,
		size:     size,
		openedAt: now,
	}, nil
}

// Write rotates the log file first if needed, optionally prepends a
// timestamp, then writes p to the current file.
func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.shouldRotate(int64(len(p))) {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	out := p
	if w.cfg.TimestampLines {
		prefix := w.now().Format("[2006-01-02 15:04:05.000] ")
		out = append([]byte(prefix), p...)
	}

	n, err := w.file.Write(out)
	w.size += int64(n)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Rotate forces an immediate rotation, subject to the configured minimum
// rotation interval.
func (w *RotatingWriter) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.withinRotationDelay() {
		return nil
	}
	return w.rotate()
}

// Close closes the current underlying file.
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

// shouldRotate reports whether a rotation should happen before writing
// incoming more bytes, honoring the minimum rotation interval.
func (w *RotatingWriter) shouldRotate(incoming int64) bool {
	if w.withinRotationDelay() {
		return false
	}
	if w.cfg.MaxBytes > 0 && w.size+incoming > w.cfg.MaxBytes {
		return true
	}
	if maxAge := w.cfg.MaxAge.Unwrap(); maxAge > 0 && w.now().Sub(w.openedAt) >= maxAge {
		return true
	}
	return false
}

// withinRotationDelay reports whether we are still inside the configured
// minimum interval since the last rotation.
func (w *RotatingWriter) withinRotationDelay() bool {
	minInterval := w.cfg.MinInterval.Unwrap()
	if minInterval <= 0 || w.lastRotation.IsZero() {
		return false
	}
	return w.now().Sub(w.lastRotation) < minInterval
}

// rotate closes the current file, renames it to a timestamped path, and
// reopens the original path for continued writing.
func (w *RotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("closing log file before rotation: %w", err)
	}

	rotatedPath := uniqueRotatedFilePath(w.path, w.now())
	if err := os.Rename(w.path, rotatedPath); err != nil {
		// Reopen the original path regardless so logging can continue even
		// if the rename failed (e.g. cross-device or permissions issue).
		f, reopenErr := openLogFile(w.path, false)
		if reopenErr == nil {
			w.file = f
		}
		return fmt.Errorf("renaming log file for rotation: %w", err)
	}

	f, err := openLogFile(w.path, false)
	if err != nil {
		return fmt.Errorf("reopening log file after rotation: %w", err)
	}

	now := w.now()
	w.file = f
	w.size = 0
	w.openedAt = now
	w.lastRotation = now
	return nil
}

// rotatedFilePath builds the timestamped filename a rotation renames the
// current log file to, e.g. "app.log" -> "app-20260619T143022.log".
func rotatedFilePath(path string, at time.Time) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s-%s%s", base, at.Format(rotationTimestampLayout), ext)
}

// uniqueRotatedFilePath returns rotatedFilePath(path, at), disambiguated
// with a numeric suffix if two rotations happen within the same second and
// would otherwise collide and silently overwrite an earlier rotated file.
func uniqueRotatedFilePath(path string, at time.Time) string {
	candidate := rotatedFilePath(path, at)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}

	ext := filepath.Ext(candidate)
	base := strings.TrimSuffix(candidate, ext)
	for i := 1; ; i++ {
		next := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(next); os.IsNotExist(err) {
			return next
		}
	}
}
