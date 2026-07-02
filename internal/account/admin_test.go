//go:build windows

package account

import "testing"

// TestIsAdminDoesNotError just verifies the real Windows token-membership
// check completes without error; whether the test runner happens to be
// elevated is environment-dependent, so we don't assert on the boolean.
func TestIsAdminDoesNotError(t *testing.T) {
	if _, err := IsAdmin(); err != nil {
		t.Fatalf("IsAdmin: unexpected error: %v", err)
	}
}

func TestRequireAdminMatchesIsAdmin(t *testing.T) {
	isAdmin, err := IsAdmin()
	if err != nil {
		t.Fatalf("IsAdmin: unexpected error: %v", err)
	}

	err = RequireAdmin()
	if isAdmin && err != nil {
		t.Errorf("RequireAdmin: expected nil when IsAdmin is true, got %v", err)
	}
	if !isAdmin && err == nil {
		t.Errorf("RequireAdmin: expected an error when IsAdmin is false")
	}
}
