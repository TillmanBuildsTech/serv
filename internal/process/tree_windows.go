//go:build windows

package process

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// startTimeImpl returns the wall-clock start time of pid via GetProcessTimes.
func startTimeImpl(pid int) (time.Time, bool) {
	creation, ok := processCreationTime(uint32(pid))
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(0, creation.Nanoseconds()), true
}

// procInfo is a lightweight snapshot of one running process, used to walk
// the process tree.
type procInfo struct {
	pid, ppid uint32
	creation  windows.Filetime
}

// killTreeImpl kills pid's descendant processes (deepest first) and then
// pid itself.
func killTreeImpl(pid int) error {
	procs, err := snapshotProcesses()
	if err != nil {
		return err
	}

	var rootCreation windows.Filetime
	if root := findProcess(procs, uint32(pid)); root != nil {
		rootCreation = root.creation
	}

	for _, child := range collectDescendants(procs, uint32(pid), rootCreation) {
		if err := terminateProcess(int(child)); err != nil {
			// Best-effort: the process may have already exited.
			_ = err
		}
	}

	return terminateProcess(pid)
}

// snapshotProcesses returns a point-in-time list of all running processes
// with their parent PID and creation time.
func snapshotProcesses() ([]procInfo, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snap, &entry); err != nil {
		return nil, err
	}

	var procs []procInfo
	for {
		creation, _ := processCreationTime(entry.ProcessID)
		procs = append(procs, procInfo{
			pid:      entry.ProcessID,
			ppid:     entry.ParentProcessID,
			creation: creation,
		})
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}

	return procs, nil
}

// processCreationTime returns the creation time of pid, or the zero value
// if it cannot be determined (e.g. the process has already exited or access
// is denied).
func processCreationTime(pid uint32) (windows.Filetime, bool) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return windows.Filetime{}, false
	}
	defer windows.CloseHandle(h)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return windows.Filetime{}, false
	}
	return creation, true
}

// findProcess returns the procInfo for pid, or nil if not present in procs.
func findProcess(procs []procInfo, pid uint32) *procInfo {
	for i := range procs {
		if procs[i].pid == pid {
			return &procs[i]
		}
	}
	return nil
}

// collectDescendants returns all descendants of parentPID, deepest first,
// so callers can kill children before parents. A candidate child is only
// included if its creation time is at or after minCreation, guarding
// against stale ParentProcessID entries left over from PID reuse after the
// real parent has already exited.
func collectDescendants(procs []procInfo, parentPID uint32, minCreation windows.Filetime) []uint32 {
	var result []uint32
	for _, p := range procs {
		if p.ppid != parentPID || p.pid == parentPID {
			continue
		}
		if filetimeToUint64(p.creation) < filetimeToUint64(minCreation) {
			continue
		}
		result = append(result, collectDescendants(procs, p.pid, p.creation)...)
		result = append(result, p.pid)
	}
	return result
}

func filetimeToUint64(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}
