//go:build windows

package platform

import (
	"errors"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Compile-time check: windowsManager implements ServiceManager.
var _ ServiceManager = (*windowsManager)(nil)

func TestWinStartType(t *testing.T) {
	cases := map[api.StartType]uint32{
		api.StartTypeAuto:    mgr.StartAutomatic,
		api.StartTypeDelayed: mgr.StartAutomatic,
		api.StartTypeManual:  mgr.StartManual,
		api.StartType(""):    mgr.StartManual,
	}
	for in, want := range cases {
		if got := winStartType(in); got != want {
			t.Errorf("winStartType(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestWinServiceStartName(t *testing.T) {
	cases := []struct {
		name string
		in   api.AccountConfig
		want string
	}{
		{"local_system", api.AccountConfig{Type: api.AccountTypeLocalSystem}, ""},
		{"empty", api.AccountConfig{}, ""},
		{"local_service", api.AccountConfig{Type: api.AccountTypeLocalService}, `NT AUTHORITY\LocalService`},
		{"network_service", api.AccountConfig{Type: api.AccountTypeNetworkService}, `NT AUTHORITY\NetworkService`},
		{"user", api.AccountConfig{Type: api.AccountTypeUser, Username: `DOMAIN\svcuser`}, `DOMAIN\svcuser`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := winServiceStartName(c.in); got != c.want {
				t.Errorf("winServiceStartName(%+v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestWinStateString(t *testing.T) {
	cases := map[svc.State]string{
		svc.Stopped:         "stopped",
		svc.StartPending:    "start_pending",
		svc.StopPending:     "stop_pending",
		svc.Running:         "running",
		svc.ContinuePending: "continue_pending",
		svc.PausePending:    "pause_pending",
		svc.Paused:          "paused",
		svc.State(999):      "unknown",
	}
	for in, want := range cases {
		if got := winStateString(in); got != want {
			t.Errorf("winStateString(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTranslateWinErr(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want string
	}{
		{"access denied", windows.ERROR_ACCESS_DENIED, "access denied"},
		{"not exist", windows.ERROR_SERVICE_DOES_NOT_EXIST, "does not exist"},
		{"exists", windows.ERROR_SERVICE_EXISTS, "already exists"},
		{"already running", windows.ERROR_SERVICE_ALREADY_RUNNING, "already running"},
		{"not active", windows.ERROR_SERVICE_NOT_ACTIVE, "not running"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := translateWinErr(c.in)
			if err == nil || !errors.Is(err, c.in) {
				t.Fatalf("translateWinErr(%v) = %v, want wrapped error", c.in, err)
			}
		})
	}

	other := errors.New("some other error")
	if got := translateWinErr(other); got != other {
		t.Errorf("translateWinErr(%v) = %v, want unchanged", other, got)
	}
}
