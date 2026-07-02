//go:build integration && linux

package integration

import (
	"os"
	"os/exec"
	"testing"
)

// requireElevated skips the calling test unless running as root with systemd
// available, since installing/removing real services requires both.
func requireElevated(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("skipping: this test installs a real service and requires running as root")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("skipping: systemctl not found (systemd may not be available in this environment)")
	}
}
