//go:build windows

package account

import (
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func TestResolveWellKnownAccounts(t *testing.T) {
	cases := []struct {
		name string
		cfg  api.AccountConfig
		want string
	}{
		{"empty type", api.AccountConfig{}, ""},
		{"local_system", api.AccountConfig{Type: api.AccountTypeLocalSystem}, ""},
		{"local_service", api.AccountConfig{Type: api.AccountTypeLocalService}, `NT AUTHORITY\LocalService`},
		{"network_service", api.AccountConfig{Type: api.AccountTypeNetworkService}, `NT AUTHORITY\NetworkService`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := Resolve(c.cfg)
			if err != nil {
				t.Fatalf("Resolve: unexpected error: %v", err)
			}
			if r.ServiceStartName != c.want {
				t.Errorf("ServiceStartName = %q, want %q", r.ServiceStartName, c.want)
			}
			if r.RequiresPassword {
				t.Errorf("RequiresPassword = true, want false")
			}
		})
	}
}

func TestResolveVirtualServiceAccount(t *testing.T) {
	cfg := api.AccountConfig{Type: api.AccountTypeUser, Username: `NT SERVICE\MyService`}
	r, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if r.ServiceStartName != `NT SERVICE\MyService` {
		t.Errorf("ServiceStartName = %q, want %q", r.ServiceStartName, `NT SERVICE\MyService`)
	}
	if r.RequiresPassword {
		t.Errorf("RequiresPassword = true, want false for virtual service account")
	}
}

func TestResolveCustomAccountRequiresPassword(t *testing.T) {
	_, err := Resolve(api.AccountConfig{Type: api.AccountTypeUser, Username: `DOMAIN\svcuser`})
	if err == nil {
		t.Fatal("Resolve: expected error when password is missing")
	}
}

func TestResolveCustomAccountWithPassword(t *testing.T) {
	cfg := api.AccountConfig{Type: api.AccountTypeUser, Username: `DOMAIN\svcuser`, Password: "hunter2"}
	r, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if r.ServiceStartName != `DOMAIN\svcuser` || !r.RequiresPassword || r.Password != "hunter2" {
		t.Errorf("Resolve = %+v, unexpected result", r)
	}
}

func TestResolveUserAccountMissingUsername(t *testing.T) {
	_, err := Resolve(api.AccountConfig{Type: api.AccountTypeUser})
	if err == nil {
		t.Fatal("Resolve: expected error for missing username")
	}
}

func TestResolveUnknownAccountType(t *testing.T) {
	_, err := Resolve(api.AccountConfig{Type: api.AccountType("bogus")})
	if err == nil {
		t.Fatal("Resolve: expected error for unknown account type")
	}
}

func TestIsVirtualServiceAccount(t *testing.T) {
	cases := map[string]bool{
		`NT SERVICE\MyService`: true,
		`nt service\MyService`: true,
		`DOMAIN\user`:          false,
		`user@domain.com`:      false,
		"":                     false,
	}
	for username, want := range cases {
		if got := IsVirtualServiceAccount(username); got != want {
			t.Errorf("IsVirtualServiceAccount(%q) = %v, want %v", username, got, want)
		}
	}
}
