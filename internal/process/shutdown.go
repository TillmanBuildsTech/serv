package process

import (
	"context"
	"fmt"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// shutdownStage is one escalation step in a graceful shutdown sequence.
type shutdownStage struct {
	name    string
	timeout time.Duration
	// run sends the shutdown signal/message for this stage. Errors are
	// non-fatal: the stage still waits out its timeout, since the process
	// may already be exiting for unrelated reasons.
	run func() error
}

type stageResult int

const (
	stageExited stageResult = iota
	stageTimedOut
	stageCancelled
)

// Shutdown runs the platform-appropriate graceful shutdown escalation for
// pid, stopping early as soon as done is closed. done should be the
// ManagedProcess's Done() channel. It returns nil once the process has
// exited, ctx.Err() if the context is cancelled mid-sequence, or an error if
// the process is still alive after every stage (including a forceful kill)
// has been attempted.
func Shutdown(ctx context.Context, pid int, done <-chan struct{}, cfg api.StopConfig) error {
	select {
	case <-done:
		return nil
	default:
	}

	return runStages(ctx, buildShutdownStages(pid, cfg), done, pid)
}

// runStages executes stages in order, waiting out each one's timeout unless
// done closes or ctx is cancelled first. It is factored out from Shutdown so
// the escalation control flow can be unit tested with synthetic stages.
func runStages(ctx context.Context, stages []shutdownStage, done <-chan struct{}, pid int) error {
	for _, stage := range stages {
		if err := stage.run(); err != nil {
			// Ignore: the process may have already exited or may not
			// support this signaling method. Still wait out the timeout in
			// case the exit is in flight.
			_ = err
		}

		switch waitStage(ctx, done, stage.timeout) {
		case stageExited:
			return nil
		case stageCancelled:
			return ctx.Err()
		case stageTimedOut:
			continue
		}
	}

	return fmt.Errorf("process %d did not exit after all shutdown stages", pid)
}

// waitStage blocks until done is closed, the context is cancelled, or
// timeout elapses, whichever happens first.
func waitStage(ctx context.Context, done <-chan struct{}, timeout time.Duration) stageResult {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return stageExited
	case <-ctx.Done():
		return stageCancelled
	case <-timer.C:
		return stageTimedOut
	}
}

// durationOrDefault returns d as a time.Duration, or def if d is unset.
func durationOrDefault(d api.Duration, def time.Duration) time.Duration {
	if d.Unwrap() <= 0 {
		return def
	}
	return d.Unwrap()
}

// methodEnabled reports whether m is present in methods, or true if methods
// is empty (meaning "use platform defaults").
func methodEnabled(methods []api.StopMethod, m api.StopMethod) bool {
	if len(methods) == 0 {
		return true
	}
	for _, x := range methods {
		if x == m {
			return true
		}
	}
	return false
}
