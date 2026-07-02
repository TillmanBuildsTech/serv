//go:build windows

package process

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
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

// TestKillTreeKillsDescendants launches a real three-generation process tree
// (root -> child -> grandchild) via the spawn_tree test helper, confirms
// all three are running, then verifies KillTree terminates every one of
// them.
func TestKillTreeKillsDescendants(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pids.txt")

	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE": "spawn_tree",
			"SPAWN_DEPTH":    "2",
			"SPAWN_PID_FILE": pidFile,
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}

	pids := waitForPIDs(t, pidFile, 3, 10*time.Second)
	if len(pids) != 3 {
		t.Fatalf("expected 3 PIDs (root + 2 descendants), got %v", pids)
	}
	if pids[0] != mp.PID {
		t.Fatalf("first recorded PID = %d, want root PID %d", pids[0], mp.PID)
	}

	for _, pid := range pids {
		if !isProcessAlive(pid) {
			t.Fatalf("pid %d should be alive before KillTree", pid)
		}
	}

	if err := KillTree(mp.PID); err != nil {
		t.Fatalf("KillTree: unexpected error: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for _, pid := range pids {
		for isProcessAlive(pid) && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if isProcessAlive(pid) {
			t.Errorf("pid %d still alive after KillTree", pid)
		}
	}
}

// TestKillTreeNoChildren verifies KillTree still kills a process with no
// descendants.
func TestKillTreeNoChildren(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE": "ignore_signals",
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}

	if err := KillTree(mp.PID); err != nil {
		t.Fatalf("KillTree: unexpected error: %v", err)
	}

	select {
	case <-mp.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit after KillTree")
	}
}

func waitForPIDs(t *testing.T, path string, n int, timeout time.Duration) []int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pids := readPIDs(path)
		if len(pids) >= n {
			return pids
		}
		time.Sleep(50 * time.Millisecond)
	}
	return readPIDs(path)
}

func readPIDs(path string) []int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var pids []int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if v, err := strconv.Atoi(scanner.Text()); err == nil {
			pids = append(pids, v)
		}
	}
	return pids
}
