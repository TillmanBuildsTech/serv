// runtimeState (and its supporting types below) is shared by the Windows
// SCM runtime (control.go) and the Linux/macOS foreground supervisor
// (run_unix.go) — none of its logic is platform-specific.
package service

import (
	"context"
	"os/exec"
	"time"

	ioredirect "github.com/TillmanBuildsTech/serv/internal/io"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/internal/hooks"
	"github.com/TillmanBuildsTech/serv/internal/process"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// stopOverallTimeout bounds the entire stop sequence (graceful shutdown
// escalation) as a last-resort safety net; process.Shutdown's own per-stage
// timeouts should normally complete well within it.
const stopOverallTimeout = 5 * time.Minute

// The following package-level vars let tests substitute fakes for the real
// process/hooks/io machinery, so runtimeState's control flow (start, stop,
// exit-action handling, backoff) can be exercised without a real SCM
// dispatcher or real Windows APIs.
var (
	startProcessFn      = process.StartProcess
	shutdownFn          = process.Shutdown
	killTreeFn          = process.KillTree
	resolveExitActionFn = process.ResolveExitAction
	hooksRunFn          = hooks.Run
	ioSetupFn           = ioredirect.Setup
)

// exitAction is the decision made after the child process exits.
type exitAction int

const (
	actionRestart exitAction = iota
	actionIgnore
	actionExit
	actionCrash
)

// runtimeState owns the supervised child process for one service instance:
// starting it, capturing its I/O, running lifecycle hooks, restarting it
// with backoff on failure, and shutting it down.
type runtimeState struct {
	name     string
	cfg      *api.ServiceConfig
	mp       *process.ManagedProcess
	redirect *ioredirect.Redirect
	throttle *process.Throttle
}

func newRuntimeState(name string, cfg *api.ServiceConfig) *runtimeState {
	return &runtimeState{
		name:     name,
		cfg:      cfg,
		throttle: process.NewThrottle(cfg.Restart),
	}
}

// done returns the current child's exit channel, or nil if there is no
// active child (e.g. after an "ignore" exit action). Receiving from a nil
// channel blocks forever, which is exactly the behavior wanted in a select:
// the caller simply stops reacting to child exit until a new child starts.
func (rt *runtimeState) done() <-chan struct{} {
	if rt.mp == nil {
		return nil
	}
	return rt.mp.Done()
}

// startChild runs the pre-start hook (which can abort the start), launches
// the child process with I/O redirection wired in, and runs the post-start
// hook.
func (rt *runtimeState) startChild() error {
	if err := hooksRunFn(rt.cfg, hooks.Context{
		ServiceName: rt.name,
		Event:       hooks.EventPreStart,
		Exe:         rt.cfg.Executable,
		Args:        rt.cfg.Arguments,
	}, 0); err != nil {
		return err
	}

	var redirect *ioredirect.Redirect
	mp, err := startProcessFn(rt.cfg, func(cmd *exec.Cmd) error {
		r, err := ioSetupFn(cmd, rt.cfg, ioredirect.Options{})
		if err != nil {
			return err
		}
		redirect = r
		return nil
	})
	if err != nil {
		return err
	}

	rt.mp = mp
	rt.redirect = redirect
	rt.throttle.RecordStart(mp.StartTime)

	_ = hooksRunFn(rt.cfg, hooks.Context{
		ServiceName: rt.name,
		PID:         mp.PID,
		Event:       hooks.EventPostStart,
		Exe:         rt.cfg.Executable,
		Args:        rt.cfg.Arguments,
	}, 0)

	return nil
}

// handleExit runs once the current child has exited: it closes the I/O
// redirection, fires the post-exit hook, and resolves the configured exit
// action for the child's exit code.
func (rt *runtimeState) handleExit() exitAction {
	mp := rt.mp
	exitCode, _ := mp.Wait()
	runtimeSeconds := int(time.Since(mp.StartTime).Seconds())

	if rt.redirect != nil {
		_ = rt.redirect.Close()
		rt.redirect = nil
	}

	_ = hooksRunFn(rt.cfg, hooks.Context{
		ServiceName:    rt.name,
		PID:            mp.PID,
		ExitCode:       exitCode,
		RuntimeSeconds: runtimeSeconds,
		Event:          hooks.EventPostExit,
		Exe:            rt.cfg.Executable,
		Args:           rt.cfg.Arguments,
	}, 0)

	switch resolveExitActionFn(rt.cfg, exitCode) {
	case api.ExitActionIgnore:
		rt.mp = nil
		return actionIgnore
	case api.ExitActionExit:
		rt.mp = nil
		return actionExit
	case api.ExitActionCrash:
		rt.mp = nil
		return actionCrash
	default:
		return actionRestart
	}
}

// nextBackoff returns how long to wait before the next restart attempt.
func (rt *runtimeState) nextBackoff() time.Duration {
	return rt.throttle.NextDelay(time.Now())
}

// stopChild runs the pre-stop hook, drives the graceful shutdown
// escalation, sweeps the process tree as a safety net, closes I/O
// redirection, and fires the post-exit hook. It blocks until the child has
// fully exited.
func (rt *runtimeState) stopChild() {
	if rt.mp == nil {
		return
	}
	mp := rt.mp

	_ = hooksRunFn(rt.cfg, hooks.Context{
		ServiceName: rt.name,
		PID:         mp.PID,
		Event:       hooks.EventPreStop,
		Exe:         rt.cfg.Executable,
		Args:        rt.cfg.Arguments,
	}, 0)

	ctx, cancel := context.WithTimeout(context.Background(), stopOverallTimeout)
	defer cancel()
	_ = shutdownFn(ctx, mp.PID, mp.Done(), rt.cfg.StopMethod)

	if rt.cfg.KillProcessTree == nil || *rt.cfg.KillProcessTree {
		_ = killTreeFn(mp.PID)
	}

	<-mp.Done()
	rt.handleExit()
}

// reload re-reads the service's on-disk configuration and, if a child is
// currently running, gracefully restarts it under the new configuration.
// Used to handle SIGHUP on Linux/macOS.
func (rt *runtimeState) reload() error {
	cfg, err := config.Load(config.DefaultConfigPath(rt.name))
	if err != nil {
		return err
	}

	if rt.mp != nil {
		rt.stopChild()
	}

	rt.cfg = cfg
	rt.throttle = process.NewThrottle(cfg.Restart)

	return rt.startChild()
}
