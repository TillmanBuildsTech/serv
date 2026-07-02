//go:build windows

package account

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// IsAdmin reports whether the current process token is a member of the
// built-in Administrators group.
func IsAdmin() (bool, error) {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false, fmt.Errorf("initializing administrators SID: %w", err)
	}
	defer windows.FreeSid(sid)

	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &token); err != nil {
		return false, fmt.Errorf("opening process token: %w", err)
	}
	defer token.Close()

	// CheckTokenMembership (which Token.IsMember wraps) requires an
	// impersonation-level token; a process's primary token must be
	// duplicated into one first.
	var impersonation windows.Token
	if err := windows.DuplicateTokenEx(
		token,
		windows.TOKEN_QUERY,
		nil,
		windows.SecurityImpersonation,
		windows.TokenImpersonation,
		&impersonation,
	); err != nil {
		return false, fmt.Errorf("duplicating process token: %w", err)
	}
	defer impersonation.Close()

	isMember, err := impersonation.IsMember(sid)
	if err != nil {
		return false, fmt.Errorf("checking administrators membership: %w", err)
	}
	return isMember, nil
}

// RequireAdmin returns a clear error if the current process is not running
// with Administrator privileges. Call this before service install/remove
// operations, which require elevation.
func RequireAdmin() error {
	isAdmin, err := IsAdmin()
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("access denied — this operation requires running as Administrator")
	}
	return nil
}
