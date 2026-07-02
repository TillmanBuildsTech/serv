//go:build windows

package process

import "golang.org/x/sys/windows"

// ignoreShutdownSignals makes the calling process immune to Ctrl+C/Ctrl+Break
// console control events, used by the "ignore_signals" test helper to
// exercise shutdown escalation. Window and thread message stages are already
// no-ops for a console-only process with no message loop.
func ignoreShutdownSignals() {
	handler := windows.NewCallback(func(ctrlType uint32) uintptr {
		return 1 // TRUE: handled, don't terminate
	})
	procSetConsoleCtrlHandler.Call(handler, 1)
}
