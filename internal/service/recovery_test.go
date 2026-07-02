//go:build windows

package service

import (
	"errors"
	"testing"
	"time"
	"unsafe"

	"github.com/TillmanBuildsTech/serv/pkg/api"
	"golang.org/x/sys/windows"
)

// fakeSCHandle is a mock scHandle used to unit test configureRecovery
// without a real SCM handle.
type fakeSCHandle struct {
	calls []uint32 // infoLevels passed to changeServiceConfig2, in order
	info  map[uint32][]byte
	err   error
}

func (f *fakeSCHandle) changeServiceConfig2(infoLevel uint32, info *byte) error {
	f.calls = append(f.calls, infoLevel)
	if f.err != nil {
		return f.err
	}
	if f.info == nil {
		f.info = make(map[uint32][]byte)
	}
	switch infoLevel {
	case windows.SERVICE_CONFIG_FAILURE_ACTIONS_FLAG:
		v := (*windows.SERVICE_FAILURE_ACTIONS_FLAG)(unsafe.Pointer(info))
		f.info[infoLevel] = []byte{byte(v.FailureActionsOnNonCrashFailures)}
	case windows.SERVICE_CONFIG_FAILURE_ACTIONS:
		v := (*windows.SERVICE_FAILURE_ACTIONS)(unsafe.Pointer(info))
		f.info[infoLevel] = []byte{byte(v.ActionsCount)}
	}
	return nil
}

func TestConfigureRecoveryDisabledOnlySetsFlag(t *testing.T) {
	h := &fakeSCHandle{}

	if err := configureRecovery(h, api.RecoveryConfig{Enabled: false}); err != nil {
		t.Fatalf("configureRecovery: unexpected error: %v", err)
	}

	if len(h.calls) != 1 || h.calls[0] != windows.SERVICE_CONFIG_FAILURE_ACTIONS_FLAG {
		t.Fatalf("expected only the failure-actions-flag call, got %v", h.calls)
	}
}

func TestConfigureRecoveryEnabledSetsFlagAndActions(t *testing.T) {
	h := &fakeSCHandle{}

	cfg := api.RecoveryConfig{
		Enabled:          true,
		FirstAction:      api.RecoveryActionRestart,
		SecondAction:     api.RecoveryActionRestart,
		SubsequentAction: api.RecoveryActionNone,
		RestartDelay:     api.Duration(5 * time.Second),
		ResetPeriod:      api.Duration(time.Hour),
	}

	if err := configureRecovery(h, cfg); err != nil {
		t.Fatalf("configureRecovery: unexpected error: %v", err)
	}

	want := []uint32{
		windows.SERVICE_CONFIG_FAILURE_ACTIONS_FLAG,
		windows.SERVICE_CONFIG_FAILURE_ACTIONS,
	}
	if len(h.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", h.calls, want)
	}
	for i := range want {
		if h.calls[i] != want[i] {
			t.Errorf("call %d = %v, want %v", i, h.calls[i], want[i])
		}
	}
}

func TestConfigureRecoveryPropagatesFlagError(t *testing.T) {
	wantErr := errors.New("access denied")
	h := &fakeSCHandle{err: wantErr}

	err := configureRecovery(h, api.RecoveryConfig{Enabled: true})
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("configureRecovery: expected wrapped error, got %v", err)
	}
}

func TestBuildActionsUsesConfiguredDelay(t *testing.T) {
	cfg := api.RecoveryConfig{
		FirstAction:      api.RecoveryActionRestart,
		SecondAction:     api.RecoveryActionRunCommand,
		SubsequentAction: api.RecoveryActionReboot,
		RestartDelay:     api.Duration(2500 * time.Millisecond),
	}

	actions := buildActions(cfg)
	if len(actions) != 3 {
		t.Fatalf("buildActions returned %d actions, want 3", len(actions))
	}

	wantTypes := []uint32{windows.SC_ACTION_RESTART, windows.SC_ACTION_RUN_COMMAND, windows.SC_ACTION_REBOOT}
	for i, want := range wantTypes {
		if actions[i].Type != want {
			t.Errorf("actions[%d].Type = %d, want %d", i, actions[i].Type, want)
		}
		if actions[i].Delay != 2500 {
			t.Errorf("actions[%d].Delay = %d, want 2500", i, actions[i].Delay)
		}
	}
}

func TestRecoveryActionTypeMapping(t *testing.T) {
	cases := map[api.RecoveryAction]uint32{
		api.RecoveryActionRestart:    windows.SC_ACTION_RESTART,
		api.RecoveryActionReboot:     windows.SC_ACTION_REBOOT,
		api.RecoveryActionRunCommand: windows.SC_ACTION_RUN_COMMAND,
		api.RecoveryActionNone:       windows.SC_ACTION_NONE,
		api.RecoveryAction(""):       windows.SC_ACTION_NONE,
		api.RecoveryAction("bogus"):  windows.SC_ACTION_NONE,
	}
	for in, want := range cases {
		if got := recoveryActionType(in); got != want {
			t.Errorf("recoveryActionType(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestBoolToInt32(t *testing.T) {
	if boolToInt32(true) != 1 {
		t.Error("boolToInt32(true) != 1")
	}
	if boolToInt32(false) != 0 {
		t.Error("boolToInt32(false) != 0")
	}
}
