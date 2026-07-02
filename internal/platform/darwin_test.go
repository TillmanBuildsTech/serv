//go:build darwin

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// Compile-time check: darwinManager implements ServiceManager.
var _ ServiceManager = (*darwinManager)(nil)

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

func withTempDaemonDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := launchDaemonDir
	launchDaemonDir = dir
	t.Cleanup(func() { launchDaemonDir = orig })
	return dir
}

func withNoopConfigSave(t *testing.T) {
	t.Helper()
	orig := writeDarwinServiceConfig
	writeDarwinServiceConfig = func(cfg *api.ServiceConfig) error { return nil }
	t.Cleanup(func() { writeDarwinServiceConfig = orig })
}

func TestDarwinManagerInstallSystemLevel(t *testing.T) {
	dir := withTempDaemonDir(t)
	withNoopConfigSave(t)

	var calls [][]string
	withMockRunCmd(t, &calls, func(name string, args ...string) (string, error) { return "", nil })

	cfg := &api.ServiceConfig{Name: "myapp", Executable: "/usr/local/bin/myapp"}
	d := &darwinManager{}
	if err := d.Install(cfg); err != nil {
		t.Fatalf("Install: unexpected error: %v", err)
	}

	plistFile := filepath.Join(dir, "com.serv.myapp.plist")
	data, err := os.ReadFile(plistFile)
	if err != nil {
		t.Fatalf("reading plist: %v", err)
	}
	if !strings.Contains(string(data), "<string>/usr/local/bin/myapp</string>") {
		t.Errorf("plist missing executable: %s", data)
	}

	if len(calls) != 1 || calls[0][1] != "load" {
		t.Fatalf("unexpected launchctl calls: %v", calls)
	}
}

func TestDarwinManagerInstallAlreadyExists(t *testing.T) {
	dir := withTempDaemonDir(t)

	if err := os.WriteFile(filepath.Join(dir, "com.serv.myapp.plist"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &darwinManager{}
	err := d.Install(&api.ServiceConfig{Name: "myapp", Executable: "/usr/local/bin/myapp"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Install: expected already-exists error, got %v", err)
	}
}

func TestDarwinManagerStartStopRestart(t *testing.T) {
	d := &darwinManager{}
	var calls [][]string
	withMockRunCmd(t, &calls, func(name string, args ...string) (string, error) { return "", nil })

	if err := d.Start("myapp"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := d.Stop("myapp"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	want := [][]string{
		{"launchctl", "start", "com.serv.myapp"},
		{"launchctl", "stop", "com.serv.myapp"},
	}
	for i := range want {
		if strings.Join(calls[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("call %d = %v, want %v", i, calls[i], want[i])
		}
	}
}

func TestDarwinManagerStatus(t *testing.T) {
	d := &darwinManager{}
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		return "{\n\t\"Label\" = \"com.serv.myapp\";\n\t\"PID\" = 4242;\n\t\"LastExitStatus\" = 0;\n};\n", nil
	})

	status, err := d.Status("myapp")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.State != "running" || status.PID != 4242 {
		t.Fatalf("Status = %+v, want running/4242", status)
	}
}

func TestDarwinManagerStatusNotFound(t *testing.T) {
	d := &darwinManager{}
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		return "Could not find service \"com.serv.myapp\" in domain for system", fmt.Errorf("exit status 113")
	})

	_, err := d.Status("myapp")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Status: expected not-found error, got %v", err)
	}
}

func TestDarwinManagerList(t *testing.T) {
	d := &darwinManager{}
	withMockRunCmd(t, nil, func(name string, args ...string) (string, error) {
		return "PID\tStatus\tLabel\n" +
			"4242\t0\tcom.serv.myapp\n" +
			"-\t0\tcom.serv.other\n" +
			"99\t0\tcom.apple.something\n", nil
	})

	list, err := d.List()
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

func TestRenderPlist(t *testing.T) {
	trueVal := true
	cfg := &api.ServiceConfig{
		Name:             "myapp",
		Executable:       "/usr/local/bin/myapp",
		Arguments:        []string{"--flag", "value"},
		WorkingDirectory: "/var/lib/myapp",
		Environment:      map[string]string{"FOO": "bar"},
		Account:          api.AccountConfig{Type: api.AccountTypeUser, Username: "svcuser"},
		Restart:          api.RestartConfig{Enabled: &trueVal},
		Stdout:           "/var/log/myapp/out.log",
	}

	plist := renderPlist(cfg)

	for _, want := range []string{
		"<key>Label</key>\n\t<string>com.serv.myapp</string>",
		"<string>/usr/local/bin/myapp</string>",
		"<string>--flag</string>",
		"<key>WorkingDirectory</key>\n\t<string>/var/lib/myapp</string>",
		"<key>FOO</key>\n\t\t<string>bar</string>",
		"<key>UserName</key>\n\t<string>svcuser</string>",
		"<key>KeepAlive</key>\n\t<true/>",
		"<key>StandardOutPath</key>\n\t<string>/var/log/myapp/out.log</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Errorf("renderPlist missing %q; got:\n%s", want, plist)
		}
	}
}

func TestRenderPlistKeepAliveDisabled(t *testing.T) {
	falseVal := false
	cfg := &api.ServiceConfig{
		Name:       "myapp",
		Executable: "/usr/local/bin/myapp",
		Restart:    api.RestartConfig{Enabled: &falseVal},
	}

	plist := renderPlist(cfg)
	if !strings.Contains(plist, "<key>KeepAlive</key>\n\t<false/>") {
		t.Errorf("renderPlist: expected KeepAlive false; got:\n%s", plist)
	}
}

func TestParsePlistInt(t *testing.T) {
	out := "{\n\t\"PID\" = 4242;\n\t\"LastExitStatus\" = 1;\n};\n"
	if got := parsePlistInt(out, "PID"); got != 4242 {
		t.Errorf("parsePlistInt(PID) = %d, want 4242", got)
	}
	if got := parsePlistInt(out, "LastExitStatus"); got != 1 {
		t.Errorf("parsePlistInt(LastExitStatus) = %d, want 1", got)
	}
	if got := parsePlistInt(out, "Missing"); got != 0 {
		t.Errorf("parsePlistInt(Missing) = %d, want 0", got)
	}
}

func TestIsUserLevel(t *testing.T) {
	if isUserLevel(&api.ServiceConfig{Account: api.AccountConfig{Type: api.AccountTypeLocalSystem}}) {
		t.Error("isUserLevel: local_system should not be user-level")
	}
	if !isUserLevel(&api.ServiceConfig{Account: api.AccountConfig{Type: api.AccountTypeUser}}) {
		t.Error("isUserLevel: user account should be user-level")
	}
}
