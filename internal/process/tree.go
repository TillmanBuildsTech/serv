package process

import "time"

// KillTree recursively kills pid and all of its descendant processes,
// killing the deepest descendants first. Implementations validate parent-
// child relationships using process start times to avoid killing an
// unrelated process that has reused a stale PID.
func KillTree(pid int) error {
	return killTreeImpl(pid)
}

// StartTime returns the wall-clock time pid was started, if it can be
// determined. It is best-effort: on some platforms the underlying source
// only has second-level (or coarser) precision.
func StartTime(pid int) (time.Time, bool) {
	return startTimeImpl(pid)
}
