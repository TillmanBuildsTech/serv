//go:build windows

// Package service configures Windows Service Control Manager behavior for
// an installed service, beyond the basic install/start/stop lifecycle
// covered by internal/platform.
package service

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/TillmanBuildsTech/serv/pkg/api"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

// scHandle abstracts the ChangeServiceConfig2 calls needed to configure
// recovery actions, so the translation logic in configureRecovery can be
// unit tested against a mock rather than a real SCM handle.
type scHandle interface {
	changeServiceConfig2(infoLevel uint32, info *byte) error
}

type winServiceHandle struct {
	handle windows.Handle
}

func (h winServiceHandle) changeServiceConfig2(infoLevel uint32, info *byte) error {
	return windows.ChangeServiceConfig2(h.handle, infoLevel, info)
}

// ConfigureRecovery applies cfg's failure-recovery actions to the named
// service via the SCM.
func ConfigureRecovery(name string, cfg api.RecoveryConfig) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connecting to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", name, err)
	}
	defer s.Close()

	return configureRecovery(winServiceHandle{handle: s.Handle}, cfg)
}

// configureRecovery drives the two ChangeServiceConfig2 calls that
// configure SCM failure recovery: the non-crash-failures flag, and (when
// recovery is enabled) the failure actions themselves.
func configureRecovery(h scHandle, cfg api.RecoveryConfig) error {
	flag := windows.SERVICE_FAILURE_ACTIONS_FLAG{
		FailureActionsOnNonCrashFailures: boolToInt32(cfg.Enabled),
	}
	if err := h.changeServiceConfig2(windows.SERVICE_CONFIG_FAILURE_ACTIONS_FLAG, (*byte)(unsafe.Pointer(&flag))); err != nil {
		return fmt.Errorf("setting failure actions flag: %w", err)
	}

	if !cfg.Enabled {
		return nil
	}

	actions := buildActions(cfg)
	fa := windows.SERVICE_FAILURE_ACTIONS{
		ResetPeriod:  uint32(cfg.ResetPeriod.Unwrap().Seconds()),
		ActionsCount: uint32(len(actions)),
		Actions:      &actions[0],
	}

	if cfg.RebootMessage != "" {
		ptr, err := syscall.UTF16PtrFromString(cfg.RebootMessage)
		if err != nil {
			return fmt.Errorf("encoding reboot message: %w", err)
		}
		fa.RebootMsg = ptr
	}
	if cfg.RunCommand != "" {
		ptr, err := syscall.UTF16PtrFromString(cfg.RunCommand)
		if err != nil {
			return fmt.Errorf("encoding run command: %w", err)
		}
		fa.Command = ptr
	}

	if err := h.changeServiceConfig2(windows.SERVICE_CONFIG_FAILURE_ACTIONS, (*byte)(unsafe.Pointer(&fa))); err != nil {
		return fmt.Errorf("setting failure actions: %w", err)
	}

	return nil
}

// buildActions builds the ordered SC_ACTION list: first, second, and
// subsequent failure actions, each using the configured restart delay.
func buildActions(cfg api.RecoveryConfig) []windows.SC_ACTION {
	delay := uint32(cfg.RestartDelay.Unwrap().Milliseconds())
	return []windows.SC_ACTION{
		{Type: recoveryActionType(cfg.FirstAction), Delay: delay},
		{Type: recoveryActionType(cfg.SecondAction), Delay: delay},
		{Type: recoveryActionType(cfg.SubsequentAction), Delay: delay},
	}
}

// recoveryActionType maps a RecoveryAction to the corresponding SC_ACTION
// type. An unset or unrecognized action is treated as SC_ACTION_NONE, so a
// service never restarts, reboots, or runs a command unless explicitly
// configured to.
func recoveryActionType(a api.RecoveryAction) uint32 {
	switch a {
	case api.RecoveryActionRestart:
		return windows.SC_ACTION_RESTART
	case api.RecoveryActionReboot:
		return windows.SC_ACTION_REBOOT
	case api.RecoveryActionRunCommand:
		return windows.SC_ACTION_RUN_COMMAND
	default:
		return windows.SC_ACTION_NONE
	}
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
