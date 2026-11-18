package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	valid := []string{"myapp", "MyApp2", "my-app", "my_app", "my.app", "0day", strings.Repeat("a", 256)}
	for _, n := range valid {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", n, err)
		}
	}
	invalid := []string{"", ".", "..", "../etc", "a/b", `a\b`, "-rf", ".hidden", "_x", "my app", "a@b", "a:b", "a\nb", "a\x00b", strings.Repeat("a", 257)}
	for _, n := range invalid {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", n)
		}
	}
}

func TestValidateControlName(t *testing.T) {
	valid := []string{"myapp", "getty@tty1", "nginx.service", "com.apple.something", "Some Service", "a:b"}
	for _, n := range valid {
		if err := ValidateControlName(n); err != nil {
			t.Errorf("ValidateControlName(%q) = %v, want nil", n, err)
		}
	}
	invalid := []string{"", ".", "..", "../etc", "a/b", `a\b`, "-rf", "--all", "a\nb", "a\x00b", "a\x7fb", strings.Repeat("a", 257)}
	for _, n := range invalid {
		if err := ValidateControlName(n); err == nil {
			t.Errorf("ValidateControlName(%q) = nil, want error", n)
		}
	}
}

func TestServiceConfigDirStaysInsideRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SERV_CONFIG_DIR", root)

	dir, err := ServiceConfigDir("myapp")
	if err != nil {
		t.Fatalf("ServiceConfigDir: %v", err)
	}
	if want := filepath.Join(root, "myapp"); dir != want {
		t.Errorf("ServiceConfigDir = %q, want %q", dir, want)
	}
	if filepath.Dir(DefaultConfigPath("myapp")) != dir {
		t.Errorf("DefaultConfigPath and ServiceConfigDir disagree")
	}
}

// Regression: `serv remove ..` used to RemoveAll(filepath.Dir(DefaultConfigPath(".."))),
// which is the *parent* of the config root (e.g. /etc on Linux).
func TestServiceConfigDirRejectsTraversal(t *testing.T) {
	t.Setenv("SERV_CONFIG_DIR", filepath.Join(t.TempDir(), "serv"))
	for _, n := range []string{"..", ".", "", "../x", "x/..", "a/b", `..\x`} {
		if dir, err := ServiceConfigDir(n); err == nil {
			t.Errorf("ServiceConfigDir(%q) = %q, want error", n, dir)
		}
	}
}

func TestValidateRejectsBadName(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse([]byte("name: \"..\"\nexecutable: " + exe + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	err = Validate(c)
	if err == nil || !strings.Contains(err.Error(), "name:") {
		t.Fatalf("Validate with name %q = %v, want a name validation error", c.Name, err)
	}
}
