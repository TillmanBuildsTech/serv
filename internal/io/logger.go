// Package io captures a child process's stdout/stderr into configured log
// files and connects a configured file to its stdin.
package io

import (
	"bytes"
	"io"
	"sync"
)

// Logger is an io.Writer that line-buffers writes before forwarding
// complete lines to an underlying destination writer. Line-buffering
// ensures that when multiple Loggers share a destination (see Redirect),
// whole lines are written atomically rather than being interleaved
// mid-line with output from another stream.
type Logger struct {
	mu  *sync.Mutex
	dst io.Writer
	buf []byte
}

// newLogger creates a Logger that writes complete lines to dst, protected
// by mu. Callers wanting stdout and stderr to share one destination file
// without interleaving pass the same *sync.Mutex to both Loggers.
func newLogger(dst io.Writer, mu *sync.Mutex) *Logger {
	return &Logger{mu: mu, dst: dst}
}

// Write buffers p and flushes any complete (newline-terminated) lines to
// the destination writer. It always reports the full length of p as
// written, matching io.Writer's contract for buffering writers.
func (l *Logger) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf = append(l.buf, p...)
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			break
		}
		line := l.buf[:i+1]
		if _, err := l.dst.Write(line); err != nil {
			return 0, err
		}
		l.buf = l.buf[i+1:]
	}

	return len(p), nil
}

// Flush writes any buffered partial line (one without a trailing newline)
// to the destination. Call it once the source is known to have no more
// data, e.g. after the child process has exited.
func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buf) == 0 {
		return nil
	}
	_, err := l.dst.Write(l.buf)
	l.buf = nil
	return err
}
