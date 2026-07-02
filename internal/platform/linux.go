//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/pkg/api"
	"gopkg.in/yaml.v3"
)

// systemdUnitDir is the directory unit files are written to. It is a
// variable so tests can redirect it to a temporary directory.
var systemdUnitDir = "/etc/systemd/system"

// runCmd executes an external command and returns its combined output. It is
// a package-level variable so tests can substitute a mock without shelling
// out to a real systemctl.
var runCmd = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// linuxManager implements ServiceManager using systemd via systemctl.
type linuxManager struct{}

func unitName(name string) string { return fmt.Sprintf("serv-%s.service", name) }
func unitPath(name string) string { return filepath.Join(systemdUnitDir, unitName(name)) }

func systemctl(args ...string) (string, error) {
	out, err := runCmd("systemctl", args...)
	if err != nil {
		return out, fmt.Errorf("systemctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out))
	}
	return out, nil
}

// Install generates a systemd unit file from cfg, registers it, and enables
// it if configured to start automatically.
func (l *linuxManager) Install(cfg *api.ServiceConfig) error {
	if cfg == nil || cfg.Name == "" {
		return fmt.Errorf("service config must have a name")
	}
	if _, err := os.Stat(unitPath(cfg.Name)); err == nil {
		return fmt.Errorf("service %q already exists", cfg.Name)
	}

	if err := os.WriteFile(unitPath(cfg.Name), []byte(renderUnit(cfg)), 0o644); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	if _, err := systemctl("daemon-reload"); err != nil {
		return err
	}

	if cfg.StartType == api.StartTypeAuto || cfg.StartType == api.StartTypeDelayed {
		if _, err := systemctl("enable", unitName(cfg.Name)); err != nil {
			return err
		}
	}

	if err := writeLinuxServiceConfig(cfg); err != nil {
		return fmt.Errorf("writing config for service %q: %w", cfg.Name, err)
	}

	return nil
}

// Remove stops, disables, and deletes the unit file for name.
func (l *linuxManager) Remove(name string) error {
	if _, err := os.Stat(unitPath(name)); os.IsNotExist(err) {
		return fmt.Errorf("service %q not found", name)
	}

	// Best-effort: the service may already be stopped/disabled.
	_, _ = systemctl("stop", unitName(name))
	_, _ = systemctl("disable", unitName(name))

	if err := os.Remove(unitPath(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}

	if _, err := systemctl("daemon-reload"); err != nil {
		return err
	}

	configDir := filepath.Dir(config.DefaultConfigPath(name))
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("removing config for service %q: %w", name, err)
	}

	return nil
}

// Start starts a stopped service via systemctl.
func (l *linuxManager) Start(name string) error {
	_, err := systemctl("start", unitName(name))
	return err
}

// Stop stops a running service via systemctl.
func (l *linuxManager) Stop(name string) error {
	_, err := systemctl("stop", unitName(name))
	return err
}

// Restart stops and starts a service via systemctl.
func (l *linuxManager) Restart(name string) error {
	_, err := systemctl("restart", unitName(name))
	return err
}

// Status queries systemd for the current state of a service.
func (l *linuxManager) Status(name string) (ServiceStatus, error) {
	out, err := systemctl("show", unitName(name), "--property=LoadState,ActiveState,SubState,MainPID,ExecMainStatus")
	if err != nil {
		return ServiceStatus{}, err
	}

	props := parseSystemdProperties(out)
	if props["LoadState"] == "not-found" {
		return ServiceStatus{}, fmt.Errorf("service %q not found", name)
	}

	pid, _ := strconv.Atoi(props["MainPID"])
	exitCode, _ := strconv.Atoi(props["ExecMainStatus"])

	return ServiceStatus{
		State:    mapSystemdState(props["ActiveState"], props["SubState"]),
		PID:      pid,
		ExitCode: exitCode,
	}, nil
}

// List returns information about all Serv-managed systemd units.
func (l *linuxManager) List() ([]ServiceInfo, error) {
	out, err := systemctl("list-units", "--type=service", "--all", "--no-legend", "--no-pager", "--plain", "serv-*.service")
	if err != nil {
		return nil, err
	}

	var infos []ServiceInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		unit := fields[0]
		active := fields[2]
		sub := fields[3]
		description := ""
		if len(fields) > 4 {
			description = strings.Join(fields[4:], " ")
		}

		name := strings.TrimSuffix(strings.TrimPrefix(unit, "serv-"), ".service")

		pid := 0
		if show, err := systemctl("show", unit, "--property=MainPID"); err == nil {
			props := parseSystemdProperties(show)
			pid, _ = strconv.Atoi(props["MainPID"])
		}

		infos = append(infos, ServiceInfo{
			Name:        name,
			DisplayName: description,
			State:       mapSystemdState(active, sub),
			PID:         pid,
		})
	}

	return infos, nil
}

// UpdateConfig regenerates the unit file and reloads systemd.
func (l *linuxManager) UpdateConfig(name string, cfg *api.ServiceConfig) error {
	if cfg == nil {
		return fmt.Errorf("service config must not be nil")
	}
	if _, err := os.Stat(unitPath(name)); os.IsNotExist(err) {
		return fmt.Errorf("service %q not found", name)
	}

	if err := os.WriteFile(unitPath(name), []byte(renderUnit(cfg)), 0o644); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	if _, err := systemctl("daemon-reload"); err != nil {
		return err
	}

	if err := writeLinuxServiceConfig(cfg); err != nil {
		return fmt.Errorf("writing config for service %q: %w", name, err)
	}

	return nil
}

// parseSystemdProperties parses `KEY=VALUE` lines as emitted by
// `systemctl show`.
func parseSystemdProperties(out string) map[string]string {
	props := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		props[k] = v
	}
	return props
}

// mapSystemdState maps systemd ActiveState/SubState to the ServiceStatus.State
// vocabulary.
func mapSystemdState(active, sub string) string {
	switch active {
	case "active":
		if sub == "running" {
			return "running"
		}
		return "running"
	case "activating":
		return "start_pending"
	case "deactivating":
		return "stop_pending"
	case "inactive":
		return "stopped"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

// renderUnit builds a systemd unit file from cfg.
func renderUnit(cfg *api.ServiceConfig) string {
	var b strings.Builder

	description := cfg.DisplayName
	if description == "" {
		description = cfg.Description
	}

	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s\n", description)
	b.WriteString("\n[Service]\n")

	execStart := cfg.Executable
	for _, arg := range cfg.Arguments {
		execStart += " " + arg
	}
	fmt.Fprintf(&b, "ExecStart=%s\n", execStart)

	if cfg.WorkingDirectory != "" {
		fmt.Fprintf(&b, "WorkingDirectory=%s\n", cfg.WorkingDirectory)
	}

	for k, v := range cfg.Environment {
		fmt.Fprintf(&b, "Environment=%s=%s\n", k, v)
	}

	if cfg.Account.Type == api.AccountTypeUser && cfg.Account.Username != "" {
		fmt.Fprintf(&b, "User=%s\n", cfg.Account.Username)
	}

	restartPolicy := "on-failure"
	if cfg.Restart.Enabled != nil && !*cfg.Restart.Enabled {
		restartPolicy = "no"
	}
	fmt.Fprintf(&b, "Restart=%s\n", restartPolicy)
	if cfg.Restart.Delay.Unwrap() > 0 {
		fmt.Fprintf(&b, "RestartSec=%d\n", int(cfg.Restart.Delay.Unwrap().Seconds()))
	}

	if cfg.Stdout != "" {
		fmt.Fprintf(&b, "StandardOutput=append:%s\n", cfg.Stdout)
	} else {
		b.WriteString("StandardOutput=journal\n")
	}
	if cfg.Stderr != "" {
		fmt.Fprintf(&b, "StandardError=append:%s\n", cfg.Stderr)
	} else {
		b.WriteString("StandardError=journal\n")
	}

	killMode := "control-group"
	if cfg.KillProcessTree != nil && !*cfg.KillProcessTree {
		killMode = "process"
	}
	fmt.Fprintf(&b, "KillMode=%s\n", killMode)

	if cfg.StopMethod.TerminateTimeout.Unwrap() > 0 {
		fmt.Fprintf(&b, "TimeoutStopSec=%d\n", int(cfg.StopMethod.TerminateTimeout.Unwrap().Seconds()))
	}

	b.WriteString("\n[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")

	return b.String()
}

// writeLinuxServiceConfig marshals cfg to YAML and writes it to the
// service's config directory, creating the directory if necessary. It is a
// variable so tests can substitute it and avoid touching /etc.
var writeLinuxServiceConfig = func(cfg *api.ServiceConfig) error {
	path := config.DefaultConfigPath(cfg.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}
