//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/pkg/api"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"gopkg.in/yaml.v3"
)

// windowsManager implements ServiceManager using the Windows Service Control
// Manager (SCM) via golang.org/x/sys/windows.
type windowsManager struct{}

// Install registers a new service with the SCM from the given configuration.
func (w *windowsManager) Install(cfg *api.ServiceConfig) error {
	if cfg == nil || cfg.Name == "" {
		return fmt.Errorf("service config must have a name")
	}

	m, err := connectSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	if existing, err := m.OpenService(cfg.Name); err == nil {
		existing.Close()
		return fmt.Errorf("service %q already exists", cfg.Name)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving serv executable path: %w", err)
	}

	mc := mgr.Config{
		DisplayName:      cfg.DisplayName,
		Description:      cfg.Description,
		StartType:        winStartType(cfg.StartType),
		DelayedAutoStart: cfg.StartType == api.StartTypeDelayed,
		ServiceStartName: winServiceStartName(cfg.Account),
		Password:         cfg.Account.Password,
	}

	s, err := m.CreateService(cfg.Name, exePath, mc, "run", cfg.Name)
	if err != nil {
		return fmt.Errorf("creating service %q: %w", cfg.Name, translateWinErr(err))
	}
	defer s.Close()

	if err := writeServiceConfig(cfg); err != nil {
		s.Delete()
		return fmt.Errorf("writing config for service %q: %w", cfg.Name, err)
	}

	return nil
}

// Remove unregisters a service from the SCM and deletes its config files.
func (w *windowsManager) Remove(name string) error {
	m, err := connectSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", name, translateWinErr(err))
	}
	defer s.Close()

	cfg, err := s.Config()
	if err != nil {
		return fmt.Errorf("reading config for service %q: %w", name, translateWinErr(err))
	}
	if !isServManaged(cfg) {
		return fmt.Errorf("service %q was not installed by serv and cannot be removed", name)
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("removing service %q: %w", name, translateWinErr(err))
	}

	configDir := filepath.Dir(config.DefaultConfigPath(name))
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("removing config for service %q: %w", name, err)
	}

	return nil
}

// Start launches a stopped service.
func (w *windowsManager) Start(name string) error {
	m, err := connectSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", name, translateWinErr(err))
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("starting service %q: %w", name, translateWinErr(err))
	}

	return nil
}

// Stop halts a running service.
func (w *windowsManager) Stop(name string) error {
	m, err := connectSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", name, translateWinErr(err))
	}
	defer s.Close()

	if _, err := s.Control(svc.Stop); err != nil {
		return fmt.Errorf("stopping service %q: %w", name, translateWinErr(err))
	}

	return nil
}

// Restart stops and then starts a service.
func (w *windowsManager) Restart(name string) error {
	if err := w.Stop(name); err != nil {
		if !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
			return err
		}
	}
	return w.Start(name)
}

// Status returns the current status of a service.
func (w *windowsManager) Status(name string) (ServiceStatus, error) {
	m, err := connectSCM()
	if err != nil {
		return ServiceStatus{}, err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("service %q not found: %w", name, translateWinErr(err))
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("querying service %q: %w", name, translateWinErr(err))
	}

	return ServiceStatus{
		State:    winStateString(status.State),
		PID:      int(status.ProcessId),
		ExitCode: int(status.Win32ExitCode),
	}, nil
}

// List returns information about all services registered with the SCM,
// including ones not installed by serv.
func (w *windowsManager) List() ([]ServiceInfo, error) {
	m, err := connectSCM()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()

	names, err := m.ListServices()
	if err != nil {
		return nil, fmt.Errorf("enumerating services: %w", translateWinErr(err))
	}

	var infos []ServiceInfo
	for _, name := range names {
		s, err := m.OpenService(name)
		if err != nil {
			continue
		}

		cfg, err := s.Config()
		if err != nil {
			s.Close()
			continue
		}

		status, err := s.Query()
		s.Close()
		if err != nil {
			continue
		}

		infos = append(infos, ServiceInfo{
			Name:        name,
			DisplayName: cfg.DisplayName,
			State:       winStateString(status.State),
			PID:         int(status.ProcessId),
		})
	}

	return infos, nil
}

// isServManaged reports whether cfg's binary path points at the currently
// running serv executable, meaning the service was installed by serv itself.
func isServManaged(cfg mgr.Config) bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(cfg.BinaryPathName, exePath)
}

// UpdateConfig applies a new configuration to an existing service.
func (w *windowsManager) UpdateConfig(name string, cfg *api.ServiceConfig) error {
	if cfg == nil {
		return fmt.Errorf("service config must not be nil")
	}

	m, err := connectSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", name, translateWinErr(err))
	}
	defer s.Close()

	current, err := s.Config()
	if err != nil {
		return fmt.Errorf("reading current config for service %q: %w", name, translateWinErr(err))
	}
	if !isServManaged(current) {
		return fmt.Errorf("service %q was not installed by serv and cannot be updated", name)
	}

	current.DisplayName = cfg.DisplayName
	current.Description = cfg.Description
	current.StartType = winStartType(cfg.StartType)
	current.DelayedAutoStart = cfg.StartType == api.StartTypeDelayed
	current.ServiceStartName = winServiceStartName(cfg.Account)
	current.Password = cfg.Account.Password

	if err := s.UpdateConfig(current); err != nil {
		return fmt.Errorf("updating service %q: %w", name, translateWinErr(err))
	}

	if err := writeServiceConfig(cfg); err != nil {
		return fmt.Errorf("writing config for service %q: %w", name, err)
	}

	return nil
}

// connectSCM opens a connection to the service control manager, returning a
// descriptive error if the caller lacks sufficient privileges.
func connectSCM() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to service control manager: %w", translateWinErr(err))
	}
	return m, nil
}

// translateWinErr wraps common Win32 service errors with clearer messages.
func translateWinErr(err error) error {
	switch {
	case errors.Is(err, windows.ERROR_ACCESS_DENIED):
		return fmt.Errorf("access denied — run as administrator: %w", err)
	case errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST):
		return fmt.Errorf("service does not exist: %w", err)
	case errors.Is(err, windows.ERROR_SERVICE_EXISTS):
		return fmt.Errorf("service already exists: %w", err)
	case errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING):
		return fmt.Errorf("service is already running: %w", err)
	case errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE):
		return fmt.Errorf("service is not running: %w", err)
	default:
		return err
	}
}

// winStartType maps a ServiceConfig start type to the corresponding SCM
// start type. Delayed auto-start is still SERVICE_AUTO_START; the delayed
// flag is applied separately via DelayedAutoStart.
func winStartType(t api.StartType) uint32 {
	switch t {
	case api.StartTypeAuto, api.StartTypeDelayed:
		return mgr.StartAutomatic
	case api.StartTypeManual:
		return mgr.StartManual
	default:
		return mgr.StartManual
	}
}

// winServiceStartName maps an AccountConfig to the SCM ServiceStartName. An
// empty string tells the SCM to use LocalSystem.
func winServiceStartName(a api.AccountConfig) string {
	switch a.Type {
	case api.AccountTypeLocalService:
		return `NT AUTHORITY\LocalService`
	case api.AccountTypeNetworkService:
		return `NT AUTHORITY\NetworkService`
	case api.AccountTypeUser:
		return a.Username
	case api.AccountTypeLocalSystem, "":
		return ""
	default:
		return ""
	}
}

// winStateString maps an svc.State to the ServiceStatus.State string vocabulary.
func winStateString(s svc.State) string {
	switch s {
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "start_pending"
	case svc.StopPending:
		return "stop_pending"
	case svc.Running:
		return "running"
	case svc.ContinuePending:
		return "continue_pending"
	case svc.PausePending:
		return "pause_pending"
	case svc.Paused:
		return "paused"
	default:
		return "unknown"
	}
}

// writeServiceConfig marshals cfg to YAML and writes it to the service's
// config directory, creating the directory if necessary.
func writeServiceConfig(cfg *api.ServiceConfig) error {
	path := config.DefaultConfigPath(cfg.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
