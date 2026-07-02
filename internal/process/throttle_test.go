package process

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func TestThrottleExponentialBackoff(t *testing.T) {
	th := NewThrottle(api.RestartConfig{
		Delay:       api.Duration(time.Millisecond),
		ThrottleCap: api.Duration(100 * time.Millisecond),
	})

	start := time.Now()
	th.RecordStart(start)

	want := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		4 * time.Millisecond,
		8 * time.Millisecond,
		16 * time.Millisecond,
		32 * time.Millisecond,
		64 * time.Millisecond,
		100 * time.Millisecond, // capped
		100 * time.Millisecond, // stays capped
	}

	when := start
	for i, w := range want {
		when = when.Add(time.Microsecond) // exits almost immediately: never triggers a reset
		got := th.NextDelay(when)
		if got != w {
			t.Fatalf("attempt %d: NextDelay = %v, want %v", i, got, w)
		}
		th.RecordStart(when)
	}
}

func TestThrottleResetsAfterSustainedUptime(t *testing.T) {
	th := NewThrottle(api.RestartConfig{
		Delay:       api.Duration(time.Millisecond),
		ThrottleCap: api.Duration(50 * time.Millisecond),
	})

	start := time.Now()
	th.RecordStart(start)

	// Two quick failures escalate the backoff.
	when := start.Add(time.Microsecond)
	if got, want := th.NextDelay(when), time.Millisecond; got != want {
		t.Fatalf("NextDelay(1) = %v, want %v", got, want)
	}
	th.RecordStart(when)

	when = when.Add(time.Microsecond)
	if got, want := th.NextDelay(when), 2*time.Millisecond; got != want {
		t.Fatalf("NextDelay(2) = %v, want %v", got, want)
	}

	// Now the process stays up for at least the throttle cap before
	// exiting again — backoff should reset to the base delay.
	sustainedStart := when
	th.RecordStart(sustainedStart)
	longRunExit := sustainedStart.Add(60 * time.Millisecond)

	if got, want := th.NextDelay(longRunExit), time.Millisecond; got != want {
		t.Fatalf("NextDelay after sustained uptime = %v, want reset to %v", got, want)
	}
}

func TestThrottleReset(t *testing.T) {
	th := NewThrottle(api.RestartConfig{Delay: api.Duration(time.Millisecond), ThrottleCap: api.Duration(100 * time.Millisecond)})
	start := time.Now()
	th.RecordStart(start)
	th.NextDelay(start.Add(time.Microsecond))
	th.NextDelay(start.Add(2 * time.Microsecond))

	th.Reset()

	if got, want := th.NextDelay(time.Now()), time.Millisecond; got != want {
		t.Fatalf("NextDelay after Reset = %v, want %v", got, want)
	}
}

func TestThrottleDefaults(t *testing.T) {
	th := NewThrottle(api.RestartConfig{})
	if th.baseDelay != defaultRestartDelay {
		t.Errorf("baseDelay = %v, want default %v", th.baseDelay, defaultRestartDelay)
	}
	if th.cap != defaultThrottleCap {
		t.Errorf("cap = %v, want default %v", th.cap, defaultThrottleCap)
	}
}

func TestWaitElapsesFully(t *testing.T) {
	start := time.Now()
	err := Wait(context.Background(), 20*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait: unexpected error: %v", err)
	}
	if elapsed < 20*time.Millisecond {
		t.Errorf("Wait returned after %v, want >= 20ms", elapsed)
	}
}

func TestWaitCancelledImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := Wait(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait: expected context.Canceled, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait took %v to cancel, want < 100ms", elapsed)
	}
}

func TestWaitCancelledMidSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := Wait(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait: expected context.Canceled, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait took %v to cancel after mid-sleep cancellation, want < 100ms", elapsed)
	}
}

func TestResolveExitActionExplicitOverride(t *testing.T) {
	cfg := &api.ServiceConfig{
		ExitActions: map[int]api.ExitAction{
			0: api.ExitActionExit,
			1: api.ExitActionRestart,
			2: api.ExitActionIgnore,
			3: api.ExitActionCrash,
		},
	}

	cases := map[int]api.ExitAction{
		0: api.ExitActionExit,
		1: api.ExitActionRestart,
		2: api.ExitActionIgnore,
		3: api.ExitActionCrash,
	}
	for code, want := range cases {
		if got := ResolveExitAction(cfg, code); got != want {
			t.Errorf("ResolveExitAction(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestResolveExitActionDefaultsToRestart(t *testing.T) {
	cfg := &api.ServiceConfig{}
	if got := ResolveExitAction(cfg, 42); got != api.ExitActionRestart {
		t.Errorf("ResolveExitAction(unmapped) = %q, want restart", got)
	}
}

func TestResolveExitActionDefaultsToExitWhenRestartDisabled(t *testing.T) {
	cfg := &api.ServiceConfig{Restart: api.RestartConfig{Enabled: api.BoolPtr(false)}}
	if got := ResolveExitAction(cfg, 42); got != api.ExitActionExit {
		t.Errorf("ResolveExitAction(unmapped, restart disabled) = %q, want exit", got)
	}
}
