//go:build windows

package account

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const seServiceLogonRight = "SeServiceLogonRight"

const (
	policyViewLocalInformation = 0x0001
	policyLookupNames          = 0x0800
	policyCreateAccount        = 0x0010
)

var (
	modadvapi32 = windows.NewLazySystemDLL("advapi32.dll")

	procLsaOpenPolicy             = modadvapi32.NewProc("LsaOpenPolicy")
	procLsaClose                  = modadvapi32.NewProc("LsaClose")
	procLsaAddAccountRights       = modadvapi32.NewProc("LsaAddAccountRights")
	procLsaEnumerateAccountRights = modadvapi32.NewProc("LsaEnumerateAccountRights")
	procLsaFreeMemory             = modadvapi32.NewProc("LsaFreeMemory")
	procLsaNtStatusToWinError     = modadvapi32.NewProc("LsaNtStatusToWinError")
)

type lsaUnicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type lsaObjectAttributes struct {
	Length                   uint32
	RootDirectory            windows.Handle
	ObjectName               *lsaUnicodeString
	Attributes               uint32
	SecurityDescriptor       uintptr
	SecurityQualityOfService uintptr
}

func newLSAUnicodeString(s string) (*lsaUnicodeString, error) {
	buf, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil, err
	}
	n := uint16(len(s) * 2)
	return &lsaUnicodeString{Length: n, MaximumLength: n, Buffer: buf}, nil
}

// lsaRights abstracts the Windows LSA calls used to check and grant the
// "Log on as a service" right, so the escalation control flow in
// EnsureServiceLogonRight can be unit tested without touching the real
// local security policy.
type lsaRights interface {
	hasServiceLogonRight(account string) (bool, error)
	grantServiceLogonRight(account string) error
}

type winLSARights struct{}

var defaultLSARights lsaRights = winLSARights{}

// EnsureServiceLogonRight grants SeServiceLogonRight to account if it does
// not already have it. Well-known and virtual service accounts should not
// be passed here — they don't need (and in the case of virtual accounts,
// can't be granted) this right.
func EnsureServiceLogonRight(account string) error {
	return ensureServiceLogonRight(defaultLSARights, account)
}

func ensureServiceLogonRight(r lsaRights, account string) error {
	has, err := r.hasServiceLogonRight(account)
	if err != nil {
		return fmt.Errorf("checking service logon right for %q: %w", account, err)
	}
	if has {
		return nil
	}
	if err := r.grantServiceLogonRight(account); err != nil {
		return fmt.Errorf("granting service logon right to %q: %w", account, err)
	}
	return nil
}

func (winLSARights) hasServiceLogonRight(account string) (bool, error) {
	sid, err := accountSID(account)
	if err != nil {
		return false, err
	}

	policy, err := openLocalPolicy(policyViewLocalInformation)
	if err != nil {
		return false, err
	}
	defer procLsaClose.Call(uintptr(policy))

	var rightsPtr unsafe.Pointer
	var count uint32
	status, _, _ := procLsaEnumerateAccountRights.Call(
		uintptr(policy),
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(&rightsPtr)),
		uintptr(unsafe.Pointer(&count)),
	)
	if status != 0 {
		// STATUS_OBJECT_NAME_NOT_FOUND means the account has no rights
		// assigned yet, which is not an error for our purposes.
		if lsaStatusIsObjectNameNotFound(uint32(status)) {
			return false, nil
		}
		return false, lsaError(uint32(status))
	}
	defer procLsaFreeMemory.Call(uintptr(rightsPtr))

	entries := unsafe.Slice((*lsaUnicodeString)(rightsPtr), count)
	for _, entry := range entries {
		if lsaUnicodeStringToString(entry) == seServiceLogonRight {
			return true, nil
		}
	}
	return false, nil
}

func (winLSARights) grantServiceLogonRight(account string) error {
	sid, err := accountSID(account)
	if err != nil {
		return err
	}

	policy, err := openLocalPolicy(policyCreateAccount | policyLookupNames)
	if err != nil {
		return err
	}
	defer procLsaClose.Call(uintptr(policy))

	right, err := newLSAUnicodeString(seServiceLogonRight)
	if err != nil {
		return err
	}

	status, _, _ := procLsaAddAccountRights.Call(
		uintptr(policy),
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(right)),
		1,
	)
	if status != 0 {
		return lsaError(uint32(status))
	}
	return nil
}

// openLocalPolicy opens the local security policy with the given access
// mask.
func openLocalPolicy(access uint32) (windows.Handle, error) {
	var attrs lsaObjectAttributes
	var policy windows.Handle

	status, _, _ := procLsaOpenPolicy.Call(
		0, // SystemName: nil targets the local system
		uintptr(unsafe.Pointer(&attrs)),
		uintptr(access),
		uintptr(unsafe.Pointer(&policy)),
	)
	if status != 0 {
		return 0, lsaError(uint32(status))
	}
	return policy, nil
}

// accountSID resolves account (e.g. "DOMAIN\user") to its SID via
// LookupAccountName.
func accountSID(account string) (*windows.SID, error) {
	var sidLen, domainLen uint32
	var use uint32

	namePtr, err := syscall.UTF16PtrFromString(account)
	if err != nil {
		return nil, err
	}

	// First call determines the required buffer sizes.
	_ = windows.LookupAccountName(nil, namePtr, nil, &sidLen, nil, &domainLen, &use)
	if sidLen == 0 {
		return nil, fmt.Errorf("looking up account %q: unable to determine SID size", account)
	}

	sid := (*windows.SID)(unsafe.Pointer(&make([]byte, sidLen)[0]))
	domain := make([]uint16, domainLen)

	if err := windows.LookupAccountName(nil, namePtr, sid, &sidLen, &domain[0], &domainLen, &use); err != nil {
		return nil, fmt.Errorf("looking up account %q: %w", account, err)
	}

	return sid, nil
}

// lsaUnicodeStringToString converts an LSA_UNICODE_STRING to a Go string.
func lsaUnicodeStringToString(s lsaUnicodeString) string {
	if s.Buffer == nil || s.Length == 0 {
		return ""
	}
	u16 := unsafe.Slice(s.Buffer, s.Length/2)
	return strings.TrimRight(windows.UTF16ToString(u16), "\x00")
}

// lsaStatusIsObjectNameNotFound reports whether status corresponds to
// STATUS_OBJECT_NAME_NOT_FOUND (0xC0000034).
func lsaStatusIsObjectNameNotFound(status uint32) bool {
	return status == 0xC0000034
}

// lsaError converts an NTSTATUS returned by an Lsa* call into a descriptive
// error using LsaNtStatusToWinError.
func lsaError(status uint32) error {
	winErr, _, _ := procLsaNtStatusToWinError.Call(uintptr(status))
	return fmt.Errorf("LSA call failed (NTSTATUS 0x%X): %w", status, syscall.Errno(winErr))
}
