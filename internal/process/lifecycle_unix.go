//go:build linux || darwin

package process

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr places the child process in its own process group so the
// full process tree can be signaled or killed independently of the parent.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
