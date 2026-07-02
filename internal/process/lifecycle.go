// Package process launches and monitors service child processes.
package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// ManagedProcess represents a launched child process and its lifecycle.
type ManagedProcess struct {
	// PID is the process identifier of the launched child.
	PID int
	// StartTime is when the child process was launched.
	StartTime time.Time

	cmd      *exec.Cmd
	exitCh   chan struct{}
	exitCode int
	exitErr  error
}

// StartOption customizes the *exec.Cmd StartProcess builds, before it is
// started. It's used to wire in stdin/stdout/stderr (see internal/io.Setup)
// without StartProcess needing to know about I/O redirection itself.
type StartOption func(cmd *exec.Cmd) error

// StartProcess launches the child process described by cfg and begins
// monitoring it for exit.
func StartProcess(cfg *api.ServiceConfig, opts ...StartOption) (*ManagedProcess, error) {
	if cfg == nil || cfg.Executable == "" {
		return nil, fmt.Errorf("service config must specify an executable")
	}

	name, args := resolveCommand(cfg.Executable, cfg.Arguments)

	cmd := exec.Command(name, args...)
	cmd.Dir = cfg.WorkingDirectory
	cmd.Env = mergeEnv(cfg.Environment)
	setSysProcAttr(cmd)

	for _, opt := range opts {
		if err := opt(cmd); err != nil {
			return nil, fmt.Errorf("configuring process %q: %w", cfg.Executable, err)
		}
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting process %q: %w", cfg.Executable, err)
	}

	mp := &ManagedProcess{
		PID:       cmd.Process.Pid,
		StartTime: time.Now(),
		cmd:       cmd,
		exitCh:    make(chan struct{}),
	}

	go mp.monitor()

	return mp, nil
}

// monitor waits for the child process to exit and records its result. It is
// the sole caller of cmd.Wait, since os/exec permits only one Wait call per
// process.
func (p *ManagedProcess) monitor() {
	err := p.cmd.Wait()
	p.exitCode = p.cmd.ProcessState.ExitCode()
	p.exitErr = err
	close(p.exitCh)
}

// Wait blocks until the child process exits and returns its exit code.
func (p *ManagedProcess) Wait() (int, error) {
	<-p.exitCh
	return p.exitCode, p.exitErr
}

// Done returns a channel that is closed when the child process exits. It
// allows callers to select on process exit alongside other events.
func (p *ManagedProcess) Done() <-chan struct{} {
	return p.exitCh
}

// Process returns the underlying os.Process for the child.
func (p *ManagedProcess) Process() *os.Process {
	return p.cmd.Process
}

// resolveCommand determines the executable and arguments to run. On
// Windows, .bat/.cmd files are wrapped with cmd.exe /C since they cannot be
// launched directly as an image.
func resolveCommand(executable string, args []string) (string, []string) {
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(executable)) {
		case ".bat", ".cmd":
			cmdArgs := append([]string{"/C", executable}, args...)
			return "cmd.exe", cmdArgs
		}
	}
	return executable, args
}

// mergeEnv combines the current process environment with service-specific
// overrides. Later entries win on duplicate keys.
func mergeEnv(extra map[string]string) []string {
	env := os.Environ()
	for k, v := range extra {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
