package io

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Options controls optional behavior of Setup.
type Options struct {
	// WriteBOM writes a UTF-8 byte-order mark at the start of any log file
	// this call creates (not one that already existed), matching Windows
	// tooling conventions for UTF-8 text files.
	WriteBOM bool
}

// Redirect owns the file handles and line-buffering loggers wired into a
// child process's stdin/stdout/stderr. Call Close after cmd.Wait returns to
// flush buffered output and release the underlying files.
type Redirect struct {
	stdinFile  *os.File
	stdoutSink io.WriteCloser
	stderrSink io.WriteCloser

	stdoutLogger *Logger
	stderrLogger *Logger
}

// Setup wires cmd.Stdin/Stdout/Stderr according to cfg. Because cmd.Stdout
// and cmd.Stderr are set to plain io.Writer values (the Loggers) rather
// than *os.File, os/exec transparently creates the anonymous pipes and the
// goroutines that copy from them; those goroutines are joined by cmd.Wait,
// so no separate goroutine management is needed here. If cfg.Stdout and
// cfg.Stderr name the same file, both streams share one destination file
// and a mutex so lines are never interleaved mid-write.
func Setup(cmd *exec.Cmd, cfg *api.ServiceConfig, opts Options) (*Redirect, error) {
	r := &Redirect{}

	if cfg.Stdin != "" {
		f, err := os.Open(cfg.Stdin)
		if err != nil {
			return nil, fmt.Errorf("opening stdin file %q: %w", cfg.Stdin, err)
		}
		r.stdinFile = f
		cmd.Stdin = f
	}

	if cfg.Stdout != "" && cfg.Stdout == cfg.Stderr {
		sink, err := newSink(cfg.Stdout, cfg, opts)
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("opening log file %q: %w", cfg.Stdout, err)
		}
		r.stdoutSink = sink

		shared := &sync.Mutex{}
		r.stdoutLogger = newLogger(sink, shared)
		r.stderrLogger = newLogger(sink, shared)
		cmd.Stdout = r.stdoutLogger
		cmd.Stderr = r.stderrLogger

		return r, nil
	}

	if cfg.Stdout != "" {
		sink, err := newSink(cfg.Stdout, cfg, opts)
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("opening stdout log file %q: %w", cfg.Stdout, err)
		}
		r.stdoutSink = sink
		r.stdoutLogger = newLogger(sink, &sync.Mutex{})
		cmd.Stdout = r.stdoutLogger
	}

	if cfg.Stderr != "" {
		sink, err := newSink(cfg.Stderr, cfg, opts)
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("opening stderr log file %q: %w", cfg.Stderr, err)
		}
		r.stderrSink = sink
		r.stderrLogger = newLogger(sink, &sync.Mutex{})
		cmd.Stderr = r.stderrLogger
	}

	return r, nil
}

// newSink opens the destination writer for path: a rotation-aware writer
// when cfg.LogRotation is enabled, or a plain append-mode log file
// otherwise.
func newSink(path string, cfg *api.ServiceConfig, opts Options) (io.WriteCloser, error) {
	if cfg.LogRotation.Enabled {
		return NewRotatingWriter(path, cfg.LogRotation)
	}
	return openLogFile(path, opts.WriteBOM)
}

// Close flushes any buffered partial lines and closes the files opened by
// Setup. Call it after cmd.Wait returns, once os/exec's internal pipe-
// copying goroutines have finished draining stdout/stderr.
func (r *Redirect) Close() error {
	var errs []error

	if r.stdoutLogger != nil {
		if err := r.stdoutLogger.Flush(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.stderrLogger != nil {
		if err := r.stderrLogger.Flush(); err != nil {
			errs = append(errs, err)
		}
	}

	if r.stdoutSink != nil {
		if err := r.stdoutSink.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.stderrSink != nil {
		if err := r.stderrSink.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.stdinFile != nil {
		if err := r.stdinFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// openLogFile opens path for appending, creating it (and its parent
// directory) if necessary. writeBOM writes a UTF-8 BOM immediately after
// creating a new (previously nonexistent or empty) file.
func openLogFile(path string, writeBOM bool) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	isNew := true
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		isNew = false
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	if writeBOM && isNew {
		if _, err := f.Write(utf8BOM); err != nil {
			f.Close()
			return nil, fmt.Errorf("writing BOM: %w", err)
		}
	}

	return f, nil
}
