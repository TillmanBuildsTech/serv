//go:build windows

package account

import (
	"errors"
	"testing"
)

// fakeLSARights is a mock lsaRights used to unit test ensureServiceLogonRight
// without touching the real local security policy.
type fakeLSARights struct {
	hasRight   bool
	hasErr     error
	grantErr   error
	hasCalls   []string
	grantCalls []string
}

func (f *fakeLSARights) hasServiceLogonRight(account string) (bool, error) {
	f.hasCalls = append(f.hasCalls, account)
	return f.hasRight, f.hasErr
}

func (f *fakeLSARights) grantServiceLogonRight(account string) error {
	f.grantCalls = append(f.grantCalls, account)
	return f.grantErr
}

func TestEnsureServiceLogonRightAlreadyGranted(t *testing.T) {
	f := &fakeLSARights{hasRight: true}

	if err := ensureServiceLogonRight(f, `DOMAIN\svcuser`); err != nil {
		t.Fatalf("ensureServiceLogonRight: unexpected error: %v", err)
	}
	if len(f.grantCalls) != 0 {
		t.Errorf("expected grantServiceLogonRight not to be called, got %v", f.grantCalls)
	}
}

func TestEnsureServiceLogonRightGrantsWhenMissing(t *testing.T) {
	f := &fakeLSARights{hasRight: false}

	if err := ensureServiceLogonRight(f, `DOMAIN\svcuser`); err != nil {
		t.Fatalf("ensureServiceLogonRight: unexpected error: %v", err)
	}
	if len(f.grantCalls) != 1 || f.grantCalls[0] != `DOMAIN\svcuser` {
		t.Errorf("expected grantServiceLogonRight called once with account, got %v", f.grantCalls)
	}
}

func TestEnsureServiceLogonRightPropagatesCheckError(t *testing.T) {
	wantErr := errors.New("lsa check failed")
	f := &fakeLSARights{hasErr: wantErr}

	err := ensureServiceLogonRight(f, `DOMAIN\svcuser`)
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("ensureServiceLogonRight: expected wrapped check error, got %v", err)
	}
	if len(f.grantCalls) != 0 {
		t.Errorf("expected grantServiceLogonRight not to be called after check error, got %v", f.grantCalls)
	}
}

func TestEnsureServiceLogonRightPropagatesGrantError(t *testing.T) {
	wantErr := errors.New("lsa grant failed")
	f := &fakeLSARights{hasRight: false, grantErr: wantErr}

	err := ensureServiceLogonRight(f, `DOMAIN\svcuser`)
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("ensureServiceLogonRight: expected wrapped grant error, got %v", err)
	}
}

func TestLsaUnicodeStringToString(t *testing.T) {
	if got := lsaUnicodeStringToString(lsaUnicodeString{}); got != "" {
		t.Errorf("lsaUnicodeStringToString(zero value) = %q, want empty", got)
	}
}

func TestLsaStatusIsObjectNameNotFound(t *testing.T) {
	if !lsaStatusIsObjectNameNotFound(0xC0000034) {
		t.Error("expected STATUS_OBJECT_NAME_NOT_FOUND to be recognized")
	}
	if lsaStatusIsObjectNameNotFound(0) {
		t.Error("expected STATUS_SUCCESS not to be recognized as not-found")
	}
}
