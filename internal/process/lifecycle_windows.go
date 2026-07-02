//go:build windows

package process

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// setSysProcAttr configures the child process to launch in its own new
// console, detached from the service process's console (if any), but
// without ever displaying a visible console window. The child still gets
// its own console session (required for shutdown.go's AttachConsole +
// GenerateConsoleCtrlEvent to deliver Ctrl+C independently of the parent);
// CREATE_NO_WINDOW just suppresses the window that CREATE_NEW_CONSOLE would
// otherwise flash on screen — noticeable and disruptive when running many
// short-lived processes, as our test suite does.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
