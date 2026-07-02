package io

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// TestMain re-executes the test binary itself as a helper process when
// GO_IO_HELPER_MODE is set, allowing tests to launch a real child process
// with known, controllable stdout/stderr/stdin behavior.
func TestMain(m *testing.M) {
	switch os.Getenv("GO_IO_HELPER_MODE") {
	case "known_output":
		fmt.Fprintln(os.Stdout, "stdout line 1")
		fmt.Fprintln(os.Stdout, "stdout line 2")
		fmt.Fprintln(os.Stderr, "stderr line 1")
		os.Exit(0)
	case "concurrent_output":
		n, _ := strconv.Atoi(os.Getenv("HELPER_LINE_COUNT"))
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				fmt.Fprintln(os.Stdout, "OUT-LINE-0123456789")
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				fmt.Fprintln(os.Stderr, "ERR-LINE-9876543210")
			}
		}()
		wg.Wait()
		os.Exit(0)
	case "echo_stdin":
		buf := make([]byte, 0)
		tmp := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}
		os.Stdout.Write(buf)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestSetupCapturesKnownOutput(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "stdout.log")
	stderrPath := filepath.Join(dir, "stderr.log")

	cfg := &api.ServiceConfig{
		Stdout: stdoutPath,
		Stderr: stderrPath,
	}

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "GO_IO_HELPER_MODE=known_output")

	r, err := Setup(cmd, cfg, Options{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Redirect.Close: %v", err)
	}

	stdoutData, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("reading stdout log: %v", err)
	}
	if got, want := string(stdoutData), "stdout line 1\nstdout line 2\n"; got != want {
		t.Fatalf("stdout log = %q, want %q", got, want)
	}

	stderrData, err := os.ReadFile(stderrPath)
	if err != nil {
		t.Fatalf("reading stderr log: %v", err)
	}
	if got, want := string(stderrData), "stderr line 1\n"; got != want {
		t.Fatalf("stderr log = %q, want %q", got, want)
	}
}

func TestSetupSameFileDoesNotInterleave(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "combined.log")

	cfg := &api.ServiceConfig{
		Stdout: logPath,
		Stderr: logPath,
	}

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "GO_IO_HELPER_MODE=concurrent_output", "HELPER_LINE_COUNT=300")

	r, err := Setup(cmd, cfg, Options{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Redirect.Close: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading combined log: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	var outCount, errCount int
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "OUT-LINE-0123456789":
			outCount++
		case "ERR-LINE-9876543210":
			errCount++
		default:
			t.Fatalf("corrupted/interleaved line: %q", line)
		}
	}
	if outCount != 300 || errCount != 300 {
		t.Fatalf("outCount=%d errCount=%d, want 300/300", outCount, errCount)
	}
}

func TestSetupStdinRedirection(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "input.txt")
	stdoutPath := filepath.Join(dir, "stdout.log")

	if err := os.WriteFile(stdinPath, []byte("hello from stdin"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &api.ServiceConfig{
		Stdin:  stdinPath,
		Stdout: stdoutPath,
	}

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "GO_IO_HELPER_MODE=echo_stdin")

	r, err := Setup(cmd, cfg, Options{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if err := cmd.Run(); err != nil {
		t.Fatalf("cmd.Run: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Redirect.Close: %v", err)
	}

	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("reading stdout log: %v", err)
	}
	if got, want := string(data), "hello from stdin"; got != want {
		t.Fatalf("echoed stdin = %q, want %q", got, want)
	}
}

func TestOpenLogFileWritesBOMOnlyForNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.log")

	f, err := openLogFile(path, true)
	if err != nil {
		t.Fatalf("openLogFile: %v", err)
	}
	f.WriteString("hello\n")
	f.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, utf8BOM) {
		t.Fatalf("expected BOM prefix, got %q", data[:min(len(data), 3)])
	}

	// Reopening an existing, non-empty file must not write a second BOM.
	f2, err := openLogFile(path, true)
	if err != nil {
		t.Fatalf("openLogFile (reopen): %v", err)
	}
	f2.WriteString("world\n")
	f2.Close()

	data2, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(data2, utf8BOM) != 1 {
		t.Fatalf("expected exactly one BOM, got data %q", data2)
	}
}

func TestOpenLogFileNoBOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nobom.log")

	f, err := openLogFile(path, false)
	if err != nil {
		t.Fatalf("openLogFile: %v", err)
	}
	f.WriteString("hello\n")
	f.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(data, utf8BOM) {
		t.Fatalf("expected no BOM, got %q", data)
	}
	if !strings.HasPrefix(string(data), "hello") {
		t.Fatalf("unexpected content: %q", data)
	}
}
