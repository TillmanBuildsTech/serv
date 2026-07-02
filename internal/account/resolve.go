//go:build windows

// Package account resolves Windows service account configuration and
// grants the "Log on as a service" right to custom accounts.
package account

import (
	"fmt"
	"strings"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// Resolved describes how a service account should be configured for the
// Windows SCM.
type Resolved struct {
	// ServiceStartName is the value to pass as CreateService/
	// ChangeServiceConfig's lpServiceStartName. An empty string tells the
	// SCM to use LocalSystem.
	ServiceStartName string
	// RequiresPassword indicates the account needs Password set when
	// creating/updating the service.
	RequiresPassword bool
	Password         string
}

// Resolve determines the SCM ServiceStartName and password requirement for
// cfg. Well-known accounts (LocalSystem/LocalService/NetworkService) and
// virtual service accounts ("NT SERVICE\<name>") never require a password;
// custom domain or local user accounts do.
func Resolve(cfg api.AccountConfig) (Resolved, error) {
	switch cfg.Type {
	case api.AccountTypeLocalSystem, "":
		return Resolved{}, nil
	case api.AccountTypeLocalService:
		return Resolved{ServiceStartName: `NT AUTHORITY\LocalService`}, nil
	case api.AccountTypeNetworkService:
		return Resolved{ServiceStartName: `NT AUTHORITY\NetworkService`}, nil
	case api.AccountTypeUser:
		return resolveUserAccount(cfg)
	default:
		return Resolved{}, fmt.Errorf("unknown account type %q", cfg.Type)
	}
}

func resolveUserAccount(cfg api.AccountConfig) (Resolved, error) {
	if cfg.Username == "" {
		return Resolved{}, fmt.Errorf("account type %q requires a username", cfg.Type)
	}

	if IsVirtualServiceAccount(cfg.Username) {
		return Resolved{ServiceStartName: cfg.Username}, nil
	}

	if cfg.Password == "" {
		return Resolved{}, fmt.Errorf("account %q requires a password", cfg.Username)
	}

	return Resolved{
		ServiceStartName: cfg.Username,
		RequiresPassword: true,
		Password:         cfg.Password,
	}, nil
}

// IsVirtualServiceAccount reports whether username names a Windows virtual
// service account, e.g. "NT SERVICE\MyService". Virtual service accounts
// are auto-created and managed by Windows and never require a password.
func IsVirtualServiceAccount(username string) bool {
	return strings.HasPrefix(strings.ToUpper(username), `NT SERVICE\`)
}
