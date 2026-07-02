//go:build windows

package hooks

import (
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
	"golang.org/x/sys/windows"
)

func isProcessAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	windows.CloseHandle(h)
	return true
}

// TestRunHookTimeoutKillsProcess verifies that a hook process which exceeds
// its deadline is actually terminated, not just abandoned.
func TestRunHookTimeoutKillsProcess(t *testing.T) {
	t.Setenv("GO_HOOK_MODE", "sleep")

	var pid int
	onHookStarted = func(p int) { pid = p }
	t.Cleanup(func() { onHookStarted = nil })

	cfg := &api.ServiceConfig{
		Hooks: map[string]string{
			"pre-stop": quotedSelf(),
		},
	}

	if err := Run(cfg, Context{Event: EventPreStop}, 200*time.Millisecond); err == nil {
		t.Fatal("Run: expected timeout error")
	}
	if pid == 0 {
		t.Fatal("onHookStarted was never called")
	}

	deadline := time.Now().Add(5 * time.Second)
	for isProcessAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if isProcessAlive(pid) {
		t.Fatalf("hook process %d still alive after timeout kill", pid)
	}
}
