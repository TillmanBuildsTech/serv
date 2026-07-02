package process

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// TestMain re-executes the test binary itself as a helper process when
// GO_HELPER_MODE is set, allowing tests to launch a real, controllable child
// process without depending on external executables.
func TestMain(m *testing.M) {
	switch os.Getenv("GO_HELPER_MODE") {
	case "exit_code":
		code, _ := strconv.Atoi(os.Getenv("HELPER_EXIT_CODE"))
		os.Exit(code)
	case "check_env_cwd":
		if os.Getenv("SERV_TEST_VAR") != "serv_test_value" {
			os.Exit(1)
		}
		wd, err := os.Getwd()
		if err != nil || wd != os.Getenv("HELPER_EXPECT_CWD") {
			os.Exit(2)
		}
		os.Exit(0)
	case "ignore_signals":
		ignoreShutdownSignals()
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "spawn_tree":
		runSpawnTree()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runSpawnTree records its own PID to SPAWN_PID_FILE, then (if SPAWN_DEPTH
// is positive) launches one more generation of itself before sleeping, so
// tests can construct a real multi-level process tree.
func runSpawnTree() {
	pidFile := os.Getenv("SPAWN_PID_FILE")
	appendPID(pidFile, os.Getpid())

	depth, _ := strconv.Atoi(os.Getenv("SPAWN_DEPTH"))
	if depth > 0 {
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(),
			"GO_HELPER_MODE=spawn_tree",
			fmt.Sprintf("SPAWN_DEPTH=%d", depth-1),
			"SPAWN_PID_FILE="+pidFile,
		)
		_ = cmd.Start()
	}

	time.Sleep(30 * time.Second)
}

func appendPID(path string, pid int) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%d\n", pid)
}

func TestStartProcessReportsPID(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE":   "exit_code",
			"HELPER_EXIT_CODE": "0",
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}
	if mp.PID <= 0 {
		t.Errorf("PID = %d, want > 0", mp.PID)
	}

	code, err := mp.Wait()
	if err != nil {
		t.Fatalf("Wait: unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestStartProcessExitCode(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE":   "exit_code",
			"HELPER_EXIT_CODE": "7",
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}

	code, err := mp.Wait()
	if err == nil {
		t.Fatal("Wait: expected non-nil error for non-zero exit code")
	}
	if code != 7 {
		t.Errorf("exit code = %d, want 7", code)
	}
}

func TestStartProcessDoneChannel(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:       "helper",
		Executable: os.Args[0],
		Environment: map[string]string{
			"GO_HELPER_MODE":   "exit_code",
			"HELPER_EXIT_CODE": "0",
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}

	select {
	case <-mp.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Done channel was not closed within timeout")
	}
}

func TestStartProcessWorkingDirectoryAndEnv(t *testing.T) {
	wd := t.TempDir()

	cfg := &api.ServiceConfig{
		Name:             "helper",
		Executable:       os.Args[0],
		WorkingDirectory: wd,
		Environment: map[string]string{
			"GO_HELPER_MODE":    "check_env_cwd",
			"SERV_TEST_VAR":     "serv_test_value",
			"HELPER_EXPECT_CWD": wd,
		},
	}

	mp, err := StartProcess(cfg)
	if err != nil {
		t.Fatalf("StartProcess: unexpected error: %v", err)
	}

	code, err := mp.Wait()
	if code != 0 {
		t.Fatalf("exit code = %d (err=%v), want 0 (working dir or env mismatch)", code, err)
	}
}

func TestStartProcessMissingExecutable(t *testing.T) {
	_, err := StartProcess(&api.ServiceConfig{Name: "helper"})
	if err == nil {
		t.Fatal("StartProcess: expected error for missing executable")
	}
}

func TestResolveCommandBatOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("bat/cmd wrapping only applies on windows")
	}

	name, args := resolveCommand(`C:\scripts\run.bat`, []string{"arg1"})
	if name != "cmd.exe" {
		t.Errorf("name = %q, want cmd.exe", name)
	}
	want := []string{"/C", `C:\scripts\run.bat`, "arg1"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestResolveCommandNonWrapped(t *testing.T) {
	name, args := resolveCommand("/usr/bin/myapp", []string{"--flag"})
	if name != "/usr/bin/myapp" {
		t.Errorf("name = %q, want /usr/bin/myapp", name)
	}
	if len(args) != 1 || args[0] != "--flag" {
		t.Errorf("args = %v, want [--flag]", args)
	}
}
