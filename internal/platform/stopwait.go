package platform

import (
	"fmt"
	"time"
)

// svcState is the subset of service states that matter when stopping a
// service and waiting for it to actually exit. It is platform-neutral so
// the stop/restart sequencing can be unit tested on any OS.
type svcState int

const (
	stateOther svcState = iota
	stateStopped
	stateStopPending
)

// stoppable is the minimal control surface needed to stop a service, wait
// for it to reach the stopped state, and start it again. windows.go adapts a
// *mgr.Service to it.
type stoppable interface {
	queryState() (svcState, error)
	sendStop() error
	start() error
}

// waitOpts bounds how long stopAndWait polls for the stopped state.
type waitOpts struct {
	timeout  time.Duration
	interval time.Duration
}

// defaultStopWait is used when no service-specific timeout can be derived.
// It comfortably exceeds the default four-stage shutdown escalation
// (4 x 1.5s) plus a pre-stop hook.
var defaultStopWait = waitOpts{timeout: 30 * time.Second, interval: 250 * time.Millisecond}

// stopAndWait asks the service to stop and blocks until the service manager
// reports it as stopped (or opts.timeout elapses). Control requests such as
// SERVICE_CONTROL_STOP are asynchronous: they return as soon as the service
// acknowledges with STOP_PENDING, long before the supervised child has gone
// through graceful shutdown. Starting again before STOPPED is reached fails
// with ERROR_SERVICE_ALREADY_RUNNING, which is what broke `serv restart`.
//
// wasRunning is false when the service was already stopped, so callers can
// decide whether that is an error (stop) or fine (restart).
func stopAndWait(s stoppable, opts waitOpts) (wasRunning bool, err error) {
	st, err := s.queryState()
	if err != nil {
		return false, err
	}

	switch st {
	case stateStopped:
		return false, nil
	case stateStopPending:
		// Already stopping (e.g. a concurrent `serv stop`); just wait.
	default:
		if err := s.sendStop(); err != nil {
			// The service may have exited between the query and the
			// control request; re-check before reporting failure.
			st2, qerr := s.queryState()
			switch {
			case qerr == nil && st2 == stateStopped:
				return false, nil
			case qerr == nil && st2 == stateStopPending:
				// Stop already in progress; fall through to waiting.
			default:
				return true, err
			}
		}
	}

	return true, waitForStopped(s, opts)
}

// waitForStopped polls until the service reports stopped or opts.timeout
// elapses.
func waitForStopped(s stoppable, opts waitOpts) error {
	if opts.interval <= 0 {
		opts.interval = defaultStopWait.interval
	}
	deadline := time.Now().Add(opts.timeout)
	for {
		st, err := s.queryState()
		if err != nil {
			return err
		}
		if st == stateStopped {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("timed out after %s waiting for the service to stop", opts.timeout)
		}
		time.Sleep(opts.interval)
	}
}

// restartService stops s (if running), waits for it to fully stop, then
// starts it.
func restartService(s stoppable, opts waitOpts) error {
	if _, err := stopAndWait(s, opts); err != nil {
		return fmt.Errorf("stopping: %w", err)
	}
	if err := s.start(); err != nil {
		return fmt.Errorf("starting: %w", err)
	}
	return nil
}
