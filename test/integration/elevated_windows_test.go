//go:build integration && windows

package integration

import (
	"testing"

	"github.com/TillmanBuildsTech/serv/internal/account"
)

// requireElevated skips the calling test unless running as Administrator,
// since installing/removing real services requires it.
func requireElevated(t *testing.T) {
	t.Helper()
	isAdmin, err := account.IsAdmin()
	if err != nil {
		t.Skipf("could not determine admin status, skipping: %v", err)
	}
	if !isAdmin {
		t.Skip("skipping: this test installs a real service and requires running as Administrator")
	}
}
