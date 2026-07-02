//go:build integration && darwin

package integration

import (
	"os"
	"testing"
)

// requireElevated skips the calling test unless running as root, since
// installing/removing real launchd services requires it.
func requireElevated(t *testing.T) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("skipping: this test installs a real service and requires running as root")
	}
}