//go:build integration && (linux || darwin)

package integration

import "syscall"

func isProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
