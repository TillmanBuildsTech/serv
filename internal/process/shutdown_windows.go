//go:build windows

package process

import (
	"time"
	"unsafe"

	"github.com/TillmanBuildsTech/serv/pkg/api"
	"golang.org/x/sys/windows"
)

const (
	defaultConsoleTimeout = 1500 * time.Millisecond
	defaultWindowTimeout  = 1500 * time.Millisecond
	defaultThreadsTimeout = 1500 * time.Millisecond
	defaultTerminateWait  = 1500 * time.Millisecond

	wmClose = 0x0010
	wmQuit  = 0x0012
)

var (
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
	moduser32   = windows.NewLazySystemDLL("user32.dll")

	procAttachConsole            = modkernel32.NewProc("AttachConsole")
	procFreeConsole              = modkernel32.NewProc("FreeConsole")
	procSetConsoleCtrlHandler    = modkernel32.NewProc("SetConsoleCtrlHandler")
	procEnumWindows              = moduser32.NewProc("EnumWindows")
	procPostMessageW             = moduser32.NewProc("PostMessageW")
	procGetWindowThreadProcessId = moduser32.NewProc("GetWindowThreadProcessId")
	procPostThreadMessageW       = moduser32.NewProc("PostThreadMessageW")
)

// buildShutdownStages returns the four-stage Windows shutdown escalation,
// matching the NSSM pattern: console Ctrl+C, window close messages, thread
// quit messages, and finally a forceful terminate. Stages absent from
// cfg.Methods (when non-empty) are skipped; TerminateProcess always runs as
// the final, unconditional fallback.
func buildShutdownStages(pid int, cfg api.StopConfig) []shutdownStage {
	var stages []shutdownStage

	if methodEnabled(cfg.Methods, api.StopMethodConsole) {
		stages = append(stages, shutdownStage{
			name:    "console",
			timeout: durationOrDefault(cfg.ConsoleTimeout, defaultConsoleTimeout),
			run:     func() error { return sendConsoleCtrlC(pid) },
		})
	}

	if methodEnabled(cfg.Methods, api.StopMethodWindow) {
		stages = append(stages, shutdownStage{
			name:    "window",
			timeout: durationOrDefault(cfg.WindowTimeout, defaultWindowTimeout),
			run:     func() error { return postCloseToWindows(pid) },
		})
	}

	if methodEnabled(cfg.Methods, api.StopMethodThreads) {
		stages = append(stages, shutdownStage{
			name:    "threads",
			timeout: durationOrDefault(cfg.ThreadsTimeout, defaultThreadsTimeout),
			run:     func() error { return postQuitToThreads(pid) },
		})
	}

	// Terminate always runs as the final, unconditional fallback so
	// shutdown eventually succeeds even under an unusual configuration.
	stages = append(stages, shutdownStage{
		name:    "terminate",
		timeout: durationOrDefault(cfg.TerminateTimeout, defaultTerminateWait),
		run:     func() error { return terminateProcess(pid) },
	})

	return stages
}

// sendConsoleCtrlC attaches to pid's console, sends CTRL_C_EVENT, then
// detaches.
func sendConsoleCtrlC(pid int) error {
	// Detach from this process's own console, if any, so AttachConsole can
	// succeed against the target process's console.
	procFreeConsole.Call()

	r, _, err := procAttachConsole.Call(uintptr(pid))
	if r == 0 {
		return err
	}
	defer procFreeConsole.Call()

	// Disable Ctrl+C handling in this process so the event isn't also
	// delivered to (and potentially terminates) the caller.
	procSetConsoleCtrlHandler.Call(0, 1)
	defer procSetConsoleCtrlHandler.Call(0, 0)

	return windows.GenerateConsoleCtrlEvent(windows.CTRL_C_EVENT, 0)
}

// postCloseToWindows posts WM_CLOSE to every top-level window owned by pid.
func postCloseToWindows(pid int) error {
	targetPID := uint32(pid)

	callback := syscallNewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var ownerPID uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&ownerPID)))
		if ownerPID == targetPID {
			procPostMessageW.Call(hwnd, wmClose, 0, 0)
		}
		return 1 // continue enumeration
	})

	procEnumWindows.Call(callback, 0)
	return nil
}

// postQuitToThreads posts WM_QUIT to every thread owned by pid.
func postQuitToThreads(pid int) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ThreadEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Thread32First(snapshot, &entry); err != nil {
		return err
	}
	for {
		if entry.OwnerProcessID == uint32(pid) {
			procPostThreadMessageW.Call(uintptr(entry.ThreadID), wmQuit, 0, 0)
		}
		if err := windows.Thread32Next(snapshot, &entry); err != nil {
			break
		}
	}

	return nil
}

// terminateProcess forcefully kills pid.
func terminateProcess(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	return windows.TerminateProcess(h, 1)
}

// syscallNewCallback wraps windows.NewCallback for the EnumWindows callback
// signature used above.
func syscallNewCallback(fn func(hwnd uintptr, lparam uintptr) uintptr) uintptr {
	return windows.NewCallback(fn)
}
