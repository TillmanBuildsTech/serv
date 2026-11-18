package platform

import (
	"errors"
	"testing"
	"time"
)

var errAlreadyRunning = errors.New("ERROR_SERVICE_ALREADY_RUNNING")

// fakeSCM models the SCM's asynchronous stop: after sendStop the service
// sits in STOP_PENDING for pendingPolls queries before reaching STOPPED,
// and start() fails unless the service is STOPPED, as StartService does.
type fakeSCM struct {
	state        svcState
	pendingPolls int
	stopErr      error
	stops        int
	starts       int
}

func (f *fakeSCM) queryState() (svcState, error) {
	if f.state == stateStopPending {
		if f.pendingPolls <= 0 {
			f.state = stateStopped
		} else {
			f.pendingPolls--
		}
	}
	return f.state, nil
}

func (f *fakeSCM) sendStop() error {
	f.stops++
	if f.stopErr != nil {
		return f.stopErr
	}
	f.state = stateStopPending
	return nil
}

func (f *fakeSCM) start() error {
	if f.state != stateStopped {
		return errAlreadyRunning
	}
	f.starts++
	f.state = stateOther
	return nil
}

var fastWait = waitOpts{timeout: time.Second, interval: time.Millisecond}

// legacyRestart reproduces the pre-fix sequence: send stop, then start
// immediately without waiting for STOPPED.
func legacyRestart(f *fakeSCM) error {
	if err := f.sendStop(); err != nil {
		return err
	}
	return f.start()
}

func TestLegacyRestartFailsWhileStopPending(t *testing.T) {
	f := &fakeSCM{state: stateOther, pendingPolls: 3}
	if err := legacyRestart(f); !errors.Is(err, errAlreadyRunning) {
		t.Fatalf("expected the old restart to hit ALREADY_RUNNING, got %v", err)
	}
}

func TestRestartWaitsForStopped(t *testing.T) {
	f := &fakeSCM{state: stateOther, pendingPolls: 5}
	if err := restartService(f, fastWait); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if f.stops != 1 || f.starts != 1 {
		t.Fatalf("stops=%d starts=%d, want 1/1", f.stops, f.starts)
	}
}

func TestRestartWhenAlreadyStoppedJustStarts(t *testing.T) {
	f := &fakeSCM{state: stateStopped}
	if err := restartService(f, fastWait); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if f.stops != 0 || f.starts != 1 {
		t.Fatalf("stops=%d starts=%d, want 0/1", f.stops, f.starts)
	}
}

func TestRestartWhileAlreadyStopPendingDoesNotResendStop(t *testing.T) {
	f := &fakeSCM{state: stateStopPending, pendingPolls: 2}
	if err := restartService(f, fastWait); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if f.stops != 0 || f.starts != 1 {
		t.Fatalf("stops=%d starts=%d, want 0/1", f.stops, f.starts)
	}
}

func TestStopAndWaitTimesOut(t *testing.T) {
	f := &fakeSCM{state: stateOther, pendingPolls: 1 << 30}
	_, err := stopAndWait(f, waitOpts{timeout: 20 * time.Millisecond, interval: time.Millisecond})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if f.starts != 0 {
		t.Fatal("must not start after a stop timeout")
	}
}

func TestStopAndWaitReportsNotRunning(t *testing.T) {
	f := &fakeSCM{state: stateStopped}
	wasRunning, err := stopAndWait(f, fastWait)
	if err != nil || wasRunning {
		t.Fatalf("wasRunning=%v err=%v, want false/nil", wasRunning, err)
	}
}

func TestStopAndWaitPropagatesControlError(t *testing.T) {
	boom := errors.New("access denied")
	f := &fakeSCM{state: stateOther, stopErr: boom}
	if _, err := stopAndWait(f, fastWait); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
}
