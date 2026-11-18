//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/internal/platform"
)

// TestServiceLifecycle drives a real install -> start -> verify running ->
// stop -> verify stopped -> remove -> verify removed cycle through the
// platform ServiceManager (the real SCM/systemd/launchd). This requires
// elevated/administrative privileges, so it skips itself when unavailable
// rather than failing CI runs that aren't elevated.
func TestServiceLifecycle(t *testing.T) {
	requireElevated(t)

	exe := helperBinary(t)
	name := uniqueServiceName(t, "lifecycle")
	cfg := baseConfig(name, exe, "-output-interval=100ms")
	cfg.DisplayName = "Serv Integration Test: " + name

	mgr := platform.NewServiceManager()

	if err := mgr.Install(cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Stop(name)
		_ = mgr.Remove(name)
	})

	if err := mgr.Start(name); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitForState(t, mgr, name, "running", 15*time.Second)

	if err := mgr.Stop(name); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	waitForState(t, mgr, name, "stopped", 15*time.Second)

	if err := mgr.Remove(name); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := mgr.Status(name); err == nil {
		t.Error("Status: expected an error for a removed service")
	}
}

// TestServiceRestart verifies that Restart on a running service brings it
// back up with a new process. Before the stop-and-wait fix, Windows Restart
// sent SERVICE_CONTROL_STOP and immediately called StartService, which
// failed with ERROR_SERVICE_ALREADY_RUNNING while the service was still
// STOP_PENDING.
func TestServiceRestart(t *testing.T) {
	requireElevated(t)

	exe := helperBinary(t)
	name := uniqueServiceName(t, "restart")
	cfg := baseConfig(name, exe, "-output-interval=100ms")

	mgr := platform.NewServiceManager()
	if err := mgr.Install(cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Stop(name)
		_ = mgr.Remove(name)
	})

	if err := mgr.Start(name); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, mgr, name, "running", 15*time.Second)
	before, err := mgr.Status(name)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if err := mgr.Restart(name); err != nil {
		t.Fatalf("Restart on a running service: %v", err)
	}
	waitForState(t, mgr, name, "running", 15*time.Second)
	after, err := mgr.Status(name)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if before.PID != 0 && after.PID == before.PID {
		t.Errorf("PID unchanged after restart (%d); service was not restarted", after.PID)
	}

	// Restart on a stopped service should simply start it.
	if err := mgr.Stop(name); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitForState(t, mgr, name, "stopped", 15*time.Second)
	if err := mgr.Restart(name); err != nil {
		t.Fatalf("Restart on a stopped service: %v", err)
	}
	waitForState(t, mgr, name, "running", 15*time.Second)
}

// TestConfigUpdateWhileStopped installs a service, updates its config while
// stopped, and confirms the update is applied without error.
func TestConfigUpdateWhileStopped(t *testing.T) {
	requireElevated(t)

	exe := helperBinary(t)
	name := uniqueServiceName(t, "configupdate")
	cfg := baseConfig(name, exe, "-output-interval=100ms")

	mgr := platform.NewServiceManager()

	if err := mgr.Install(cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Remove(name) })

	cfg.Description = "updated description"
	if err := mgr.UpdateConfig(name, cfg); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}
}

func waitForState(t *testing.T, mgr platform.ServiceManager, name, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		status, err := mgr.Status(name)
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		last = status.State
		if status.State == want {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("service %q did not reach state %q within %v (last observed: %q)", name, want, timeout, last)
}
