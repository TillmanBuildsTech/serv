package process

import (
	"context"
	"sync"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

const (
	defaultRestartDelay = time.Second
	defaultThrottleCap  = 60 * time.Second
)

// Throttle tracks restart attempts for a single service and computes the
// exponential backoff delay before the next restart. Backoff resets once
// the process has stayed up for at least the throttle cap duration.
type Throttle struct {
	baseDelay time.Duration
	cap       time.Duration

	mu        sync.Mutex
	attempt   int
	lastStart time.Time
}

// NewThrottle creates a Throttle from a service's restart configuration.
// RestartConfig.Delay is used as the base backoff (default 1s), and
// ThrottleCap caps the maximum backoff and defines the sustained-uptime
// threshold that resets it (default 60s).
func NewThrottle(cfg api.RestartConfig) *Throttle {
	return &Throttle{
		baseDelay: durationOrDefault(cfg.Delay, defaultRestartDelay),
		cap:       durationOrDefault(cfg.ThrottleCap, defaultThrottleCap),
	}
}

// RecordStart marks the moment a (re)started process began running. Call
// this immediately after successfully launching the child process.
func (t *Throttle) RecordStart(when time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastStart = when
}

// NextDelay computes the backoff to wait before the next restart attempt,
// given the process exited at "when". If the process ran for at least the
// throttle cap duration since its last recorded start, the backoff resets
// to the base delay.
func (t *Throttle) NextDelay(when time.Time) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.lastStart.IsZero() && when.Sub(t.lastStart) >= t.cap {
		t.attempt = 0
	}

	delay := t.baseDelay
	for i := 0; i < t.attempt && delay < t.cap; i++ {
		delay *= 2
	}
	if delay > t.cap {
		delay = t.cap
	}

	t.attempt++
	return delay
}

// Reset clears all backoff state, as if the throttle were newly created.
func (t *Throttle) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.attempt = 0
	t.lastStart = time.Time{}
}

// Wait blocks for delay, or until ctx is cancelled, whichever comes first.
// It returns nil if the full delay elapsed, or ctx.Err() if cancelled early
// — used to make restart backoff immediately interruptible by a stop
// command.
func Wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ResolveExitAction determines what should happen after the service's
// process exits with the given code. An explicit per-exit-code override in
// cfg.ExitActions always wins; otherwise it falls back to Restart, unless
// restarting is disabled entirely (Restart.Enabled == false), in which case
// it falls back to Exit.
func ResolveExitAction(cfg *api.ServiceConfig, exitCode int) api.ExitAction {
	if action, ok := cfg.ExitActions[exitCode]; ok {
		return action
	}
	if cfg.Restart.Enabled != nil && !*cfg.Restart.Enabled {
		return api.ExitActionExit
	}
	return api.ExitActionRestart
}
