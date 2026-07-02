//go:build darwin

package process

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// startTimeImpl returns the wall-clock start time of pid via `ps -o lstart=`.
func startTimeImpl(pid int) (time.Time, bool) {
	return processStartTime(pid)
}

// killTreeImpl kills pid's descendant processes (deepest first, discovered
// via pgrep -P), then sweeps the process group as a safety net for any
// reparented stragglers, then kills pid itself.
func killTreeImpl(pid int) error {
	startTime, _ := processStartTime(pid)

	for _, child := range collectDescendantsDarwin(pid, startTime) {
		if err := syscall.Kill(child, syscall.SIGKILL); err != nil {
			_ = err // best-effort: the process may have already exited
		}
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)

	return syscall.Kill(pid, syscall.SIGKILL)
}

// collectDescendantsDarwin returns all descendants of pid, deepest first,
// so callers can kill children before parents. A candidate child is only
// included if its start time is at or after minStartTime, guarding against
// PID reuse after the real parent has already exited. A zero minStartTime
// disables this check.
func collectDescendantsDarwin(pid int, minStartTime time.Time) []int {
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		return nil
	}

	var result []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		childPID, err := strconv.Atoi(line)
		if err != nil {
			continue
		}

		childStart, ok := processStartTime(childPID)
		if !minStartTime.IsZero() && ok && childStart.Before(minStartTime) {
			continue
		}

		result = append(result, collectDescendantsDarwin(childPID, childStart)...)
		result = append(result, childPID)
	}

	return result
}

// processStartTime returns pid's start time via `ps -o lstart=`, or the
// zero time if it cannot be determined.
func processStartTime(pid int) (time.Time, bool) {
	out, err := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return time.Time{}, false
	}

	t, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", strings.TrimSpace(string(out)), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
