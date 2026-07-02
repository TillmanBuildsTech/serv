//go:build integration

package integration

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/internal/process"
)

// TestProcessTreeKilling launches the helper fixture configured to spawn
// one child, confirms both processes are running, then verifies
// process.KillTree terminates the entire tree.
func TestProcessTreeKilling(t *testing.T) {
	exe := helperBinary(t)
	pidFile := filepath.Join(t.TempDir(), "pids.txt")

	cfg := baseConfig("treekill", exe,
		"-spawn-child",
		"-pid-file", pidFile,
		"-output-interval=50ms",
	)

	mp, err := process.StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}

	pids := waitForPIDCount(t, pidFile, 2, 10*time.Second)
	if len(pids) != 2 {
		t.Fatalf("expected 2 PIDs (parent + child), got %v", pids)
	}

	for _, pid := range pids {
		if !isProcessAlive(pid) {
			t.Fatalf("pid %d should be alive before KillTree", pid)
		}
	}

	if err := process.KillTree(mp.PID); err != nil {
		t.Fatalf("KillTree: %v", err)
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

func waitForPIDCount(t *testing.T, path string, n int, timeout time.Duration) []int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pids := readPIDFile(path)
		if len(pids) >= n {
			return pids
		}
		time.Sleep(50 * time.Millisecond)
	}
	return readPIDFile(path)
}

func readPIDFile(path string) []int {
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
