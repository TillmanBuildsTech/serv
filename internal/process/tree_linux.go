//go:build linux

package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// clockTicksPerSecond is the conventional Linux value (USER_HZ), used to
// convert /proc/<pid>/stat's starttime field (in clock ticks since boot)
// into a duration. It is not queryable without cgo, but 100 is correct on
// the overwhelming majority of Linux systems.
const clockTicksPerSecond = 100

// startTimeImpl returns the wall-clock start time of pid, computed from its
// /proc/<pid>/stat starttime (ticks since boot) plus the system boot time
// from /proc/stat.
func startTimeImpl(pid int) (time.Time, bool) {
	ticks, ok := processStartTime(pid)
	if !ok {
		return time.Time{}, false
	}

	boot, ok := bootTime()
	if !ok {
		return time.Time{}, false
	}

	return boot.Add(time.Duration(ticks) * time.Second / clockTicksPerSecond), true
}

// bootTime reads the system boot time from the "btime" line of /proc/stat.
func bootTime() (time.Time, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "btime" {
			sec, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return time.Time{}, false
			}
			return time.Unix(sec, 0), true
		}
	}
	return time.Time{}, false
}

// killTreeImpl kills pid's descendant processes (deepest first, discovered
// via /proc/<pid>/children), then sweeps the process group as a safety net
// for any reparented stragglers, then kills pid itself.
func killTreeImpl(pid int) error {
	startTime, _ := processStartTime(pid)

	for _, child := range collectDescendantsLinux(pid, startTime) {
		if err := syscall.Kill(child, syscall.SIGKILL); err != nil {
			_ = err // best-effort: the process may have already exited
		}
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)

	return syscall.Kill(pid, syscall.SIGKILL)
}

// collectDescendantsLinux returns all descendants of pid, deepest first, so
// callers can kill children before parents. A candidate child is only
// included if its start time is at or after minStartTime, guarding against
// stale /proc entries left over from PID reuse after the real parent has
// already exited. minStartTime of 0 disables this check.
func collectDescendantsLinux(pid int, minStartTime uint64) []int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/children", pid))
	if err != nil {
		return nil
	}

	var result []int
	for _, f := range strings.Fields(string(data)) {
		childPID, err := strconv.Atoi(f)
		if err != nil {
			continue
		}

		childStart, ok := processStartTime(childPID)
		if minStartTime != 0 && ok && childStart < minStartTime {
			continue
		}

		result = append(result, collectDescendantsLinux(childPID, childStart)...)
		result = append(result, childPID)
	}

	return result
}

// processStartTime returns pid's start time as reported in field 22 of
// /proc/<pid>/stat (clock ticks since boot), or 0 if it cannot be
// determined.
func processStartTime(pid int) (uint64, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}

	// The process name field is parenthesized and may itself contain
	// spaces or parentheses, so skip past its closing paren before
	// splitting the remaining whitespace-delimited fields.
	s := string(data)
	idx := strings.LastIndex(s, ")")
	if idx < 0 {
		return 0, false
	}

	fields := strings.Fields(s[idx+1:])
	if len(fields) < 20 {
		return 0, false
	}

	// fields[19] is the 22nd field overall (starttime), since fields[0]
	// corresponds to field 3 (state).
	startTime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, false
	}
	return startTime, true
}
