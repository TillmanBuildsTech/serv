// Package hooks executes user-configured external commands at service
// lifecycle events (pre-start, post-start, pre-stop, post-exit, rotate).
package hooks

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/TillmanBuildsTech/serv/internal/process"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// DefaultTimeout is the hook execution deadline used when none is
// configured.
const DefaultTimeout = 60 * time.Second

// onHookStarted, when non-nil, is called with the PID of each hook process
// immediately after it starts. It exists solely so tests can observe and
// verify the process Run actually launched and killed.
var onHookStarted func(pid int)

// Run executes the hook configured for ctx.Event, if any, via the system
// shell, and waits for it to complete or timeout to elapse, whichever comes
// first. It returns nil if no hook is configured for the event. A hook
// that exits non-zero, or that exceeds timeout (in which case its process
// tree is killed), returns an error — callers use this to decide whether to
// abort the action the hook is guarding (e.g. pre-start aborting service
// start).
func Run(cfg *api.ServiceConfig, ctx Context, timeout time.Duration) error {
	command, ok := cfg.Hooks[string(ctx.Event)]
	if !ok || command == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	cmd := buildCommand(command)
	cmd.Env = buildEnv(ctx)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s hook: %w", ctx.Event, err)
	}
	if onHookStarted != nil {
		onHookStarted(cmd.Process.Pid)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("%s hook failed: %w", ctx.Event, err)
		}
		return nil
	case <-time.After(timeout):
		_ = process.KillTree(cmd.Process.Pid)
		<-done // reap the process so cmd.Wait's goroutine doesn't leak
		return fmt.Errorf("%s hook exceeded %s deadline and was killed", ctx.Event, timeout)
	}
}

// buildCommand wraps command in the platform's shell, so hook
// configuration can use shell syntax (pipes, redirection, etc.) just like
// NSSM-style hook scripts.
func buildCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd.exe", "/C", command)
	}
	return exec.Command("/bin/sh", "-c", command)
}
