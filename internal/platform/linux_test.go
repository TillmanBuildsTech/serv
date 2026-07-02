//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// Compile-time check: linuxManager implements ServiceManager.
var _ ServiceManager = (*linuxManager)(nil)

// withMockRunCmd swaps runCmd for the duration of the test.
func withMockRunCmd(t *testing.T, calls *[][]string, fn func(name string, args ...string) (string, error)) {
	t.Helper()
	orig := runCmd
	runCmd = func(name string, args ...string) (string, error) {
		if calls != nil {
			*calls = append(*calls, append([]string{name}, args...))
		}
		return fn(name, args...)
	}
	t.Cleanup(func() { runCmd = orig })
}

func TestLinuxManagerInstall(t *testing.T) {
	dir := t.TempDir()
	origDir := systemdUnitDir
	systemdUnitDir = dir
	t.Cleanup(func() { systemdUnitDir = origDir })

	origSave := writeLinuxServiceConfig
	writeLinuxServiceConfig = func(cfg *api.ServiceConfig) error { return nil }
	t.Cleanup(func() { writeLinuxServiceConfig = origSave })

	var calls [][]string
	withMockRunCmd(t, &calls, func(name string, args ...string) (string, error) {
		return "", nil
	})

	cfg := &api.ServiceConfig{
		Name:       "myapp",
		Executable: "/usr/bin/myapp",
		StartType:  api.StartTypeAuto,
	}

	l := &linuxManager{}
	if err := l.Install(cfg); err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}

	unitFile := filepath.Join(dir, "serv-myapp.service")
	data, err := os.ReadFile(unitFile)
	if err != nil {
		t.Fatalf("reading unit file: %v", err)
	}
	if !strings.Contains(string(data), "ExecStart=/usr/bin/myapp") {
		t.Errorf("unit file missing ExecStart: %s", data)
	}

	if len(calls) != 2 || calls[0][1] != "daemon-reload" || calls[1][1] != "enable" {
		t.Fatalf("unexpected systemctl calls: %v", calls)
	}
}

func TestLinuxManagerInstallAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	origDir := systemdUnitDir
	systemdUnitDir = dir
	t.Cleanup(func() { systemdUnitDir = origDir })

	if err := os.WriteFile(filepath.Join(dir, "serv-myapp.service"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	l := &linuxManager{}
	err := l.Install(&api.ServiceConfig{Name: "myapp", Executable: "/usr/bin/myapp"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Install: expected already-exists error, got %v", err)
	}
}

func TestLinuxManagerStartStopRestart(t *testing.T) {
	l := &linuxManager{}

	var calls [][]string
	withMockRunCmd(t, &calls, func(name string, args ...string) (string, error) {
		return "", nil
	})

	if err := l.Start("myapp"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := l.Stop("myapp"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := l.Restart("myapp"); err != nil {
		t.Fatalf("Restart: %v", err)
	}

	want := [][]string{
		{"systemctl", "start", "serv-myapp.service"},
		{"systemctl", "stop", "serv-myapp.service"},
		{"systemctl", "restart", "serv-myapp.service"},
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	for i := range want {
		if strings.Join(calls[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("call %d = %v, want %v", i, calls[i], want[i])
		}
	}
}

func TestLinuxManagerStartError(t *testing.T) {
	l := &linuxManager{}
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		return "Unit serv-myapp.service not found.", fmt.Errorf("exit status 5")
	})

	err := l.Start("myapp")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Start: expected wrapped error, got %v", err)
	}
}

func TestLinuxManagerStatus(t *testing.T) {
	l := &linuxManager{}
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		return "LoadState=loaded\nActiveState=active\nSubState=running\nMainPID=4242\nExecMainStatus=0\n", nil
	})

	status, err := l.Status("myapp")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.State != "running" || status.PID != 4242 {
		t.Fatalf("Status = %+v, want running/4242", status)
	}
}

func TestLinuxManagerStatusNotFound(t *testing.T) {
	l := &linuxManager{}
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		return "LoadState=not-found\nActiveState=inactive\nSubState=dead\nMainPID=0\nExecMainStatus=0\n", nil
	})

	_, err := l.Status("myapp")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Status: expected not-found error, got %v", err)
	}
}

func TestLinuxManagerList(t *testing.T) {
	l := &linuxManager{}
	call := 0
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		call++
		if call == 1 {
			return "serv-myapp.service loaded active running My App\n" +
				"serv-other.service loaded inactive dead Other App\n", nil
		}
		return "MainPID=4242\n", nil
	})

	list, err := l.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List: got %d entries, want 2", len(list))
	}
	if list[0].Name != "myapp" || list[0].State != "running" || list[0].PID != 4242 {
		t.Errorf("List[0] = %+v", list[0])
	}
	if list[1].Name != "other" || list[1].State != "stopped" {
		t.Errorf("List[1] = %+v", list[1])
	}
}

func TestRenderUnit(t *testing.T) {
	trueVal := true
	cfg := &api.ServiceConfig{
		Name:             "myapp",
		DisplayName:      "My App",
		Executable:       "/usr/bin/myapp",
		Arguments:        []string{"--flag", "value"},
		WorkingDirectory: "/var/lib/myapp",
		Environment:      map[string]string{"FOO": "bar"},
		Account:          api.AccountConfig{Type: api.AccountTypeUser, Username: "svcuser"},
		Restart:          api.RestartConfig{Enabled: &trueVal},
		Stdout:           "/var/log/myapp/out.log",
		KillProcessTree:  &trueVal,
	}

	unit := renderUnit(cfg)

	for _, want := range []string{
		"Description=My App",
		"ExecStart=/usr/bin/myapp --flag value",
		"WorkingDirectory=/var/lib/myapp",
		"Environment=FOO=bar",
		"User=svcuser",
		"Restart=on-failure",
		"StandardOutput=append:/var/log/myapp/out.log",
		"KillMode=control-group",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("renderUnit missing %q; got:\n%s", want, unit)
		}
	}
}

func TestRenderUnitRestartDisabled(t *testing.T) {
	falseVal := false
	cfg := &api.ServiceConfig{
		Name:       "myapp",
		Executable: "/usr/bin/myapp",
		Restart:    api.RestartConfig{Enabled: &falseVal},
	}

	unit := renderUnit(cfg)
	if !strings.Contains(unit, "Restart=no") {
		t.Errorf("renderUnit: expected Restart=no; got:\n%s", unit)
	}
}

func TestMapSystemdState(t *testing.T) {
	cases := []struct {
		active, sub, want string
	}{
		{"active", "running", "running"},
		{"activating", "auto-restart", "start_pending"},
		{"deactivating", "stop-sigterm", "stop_pending"},
		{"inactive", "dead", "stopped"},
		{"failed", "failed", "failed"},
		{"reloading", "reload", "unknown"},
	}
	for _, c := range cases {
		if got := mapSystemdState(c.active, c.sub); got != c.want {
			t.Errorf("mapSystemdState(%q, %q) = %q, want %q", c.active, c.sub, got, c.want)
		}
	}
}

func TestParseSystemdProperties(t *testing.T) {
	out := "LoadState=loaded\nActiveState=active\nSubState=running\n\n"
	props := parseSystemdProperties(out)
	if props["LoadState"] != "loaded" || props["ActiveState"] != "active" || props["SubState"] != "running" {
		t.Errorf("parseSystemdProperties = %v", props)
	}
}
