package io

import (
	"bytes"
	"sync"
	"testing"
)

func TestLoggerBuffersUntilNewline(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf, &sync.Mutex{})

	if _, err := l.Write([]byte("hello ")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("buf = %q before newline, want empty", got)
	}

	if _, err := l.Write([]byte("world\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buf.String(); got != "hello world\n" {
		t.Fatalf("buf = %q, want %q", got, "hello world\n")
	}
}

func TestLoggerMultipleLinesInOneWrite(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf, &sync.Mutex{})

	if _, err := l.Write([]byte("line1\nline2\nline3")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got, want := buf.String(), "line1\nline2\n"; got != want {
		t.Fatalf("buf = %q, want %q", got, want)
	}

	if err := l.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if got, want := buf.String(), "line1\nline2\nline3"; got != want {
		t.Fatalf("buf after flush = %q, want %q", got, want)
	}
}

func TestLoggerFlushNoop(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf, &sync.Mutex{})

	if _, err := l.Write([]byte("complete\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := l.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if got, want := buf.String(), "complete\n"; got != want {
		t.Fatalf("buf = %q, want %q", got, want)
	}
}

func TestLoggerWriteReturnsFullLength(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf, &sync.Mutex{})

	p := []byte("partial without newline")
	n, err := l.Write(p)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(p) {
		t.Fatalf("Write returned n=%d, want %d", n, len(p))
	}
}

// TestLoggerSharedMutexPreventsInterleaving simulates two Loggers (as used
// for stdout+stderr pointing at the same destination) writing many complete
// lines concurrently and verifies every line lands intact.
func TestLoggerSharedMutexPreventsInterleaving(t *testing.T) {
	var buf syncBuffer
	mu := &sync.Mutex{}
	out := newLogger(&buf, mu)
	err := newLogger(&buf, mu)

	const n = 500
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			out.Write([]byte("OUT-LINE-0123456789\n"))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			err.Write([]byte("ERR-LINE-9876543210\n"))
		}
	}()
	wg.Wait()

	lines := bytes.Split(bytes.TrimRight(buf.Bytes(), "\n"), []byte("\n"))
	if len(lines) != 2*n {
		t.Fatalf("got %d lines, want %d", len(lines), 2*n)
	}
	for _, line := range lines {
		s := string(line)
		if s != "OUT-LINE-0123456789" && s != "ERR-LINE-9876543210" {
			t.Fatalf("corrupted/interleaved line: %q", s)
		}
	}
}

// syncBuffer wraps bytes.Buffer with a mutex so tests can safely read it
// while goroutines might still be finishing writes elsewhere.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}
