//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/internal/hooks"
	"github.com/TillmanBuildsTech/serv/internal/process"
)

// TestPreStartHookAbort verifies that a failing pre-start hook prevents the
// guarded process from being considered started, matching how the service
// runtime uses hooks.Run to gate startChild.
func TestPreStartHookAbort(t *testing.T) {
	cfg := baseConfig("hookabort", helperBinary(t), "-exit-after=5s")
	cfg.Hooks = map[string]string{
		"pre-start": failingHookCommand(),
	}

	err := hooks.Run(cfg, hooks.Context{Event: hooks.EventPreStart}, 5*time.Second)
	if err == nil {
		t.Fatal("hooks.Run: expected error for failing pre-start hook")
	}
}

// TestPreStartHookAllowsStart verifies a successful pre-start hook does not
// block the subsequent process start.
func TestPreStartHookAllowsStart(t *testing.T) {
	cfg := baseConfig("hookok", helperBinary(t), "-exit-after=300ms")
	cfg.Hooks = map[string]string{
		"pre-start": succeedingHookCommand(),
	}

	if err := hooks.Run(cfg, hooks.Context{Event: hooks.EventPreStart}, 5*time.Second); err != nil {
		t.Fatalf("hooks.Run: unexpected error: %v", err)
	}

	mp, err := process.StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	select {
	case <-mp.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("helper did not exit within timeout")
	}
}

// hooks.Run already wraps the command in the platform shell (cmd.exe /C on
// Windows, /bin/sh -c elsewhere), so "exit N" works unmodified on both,
// since it's a builtin in both cmd.exe and POSIX shells.
func failingHookCommand() string    { return "exit 1" }
func succeedingHookCommand() string { return "exit 0" }
