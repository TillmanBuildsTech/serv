//go:build darwin

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

// launchDaemonDir and launchAgentDir hold the plist directories. They are
// variables so tests can redirect them to a temporary directory.
var (
	launchDaemonDir = "/Library/LaunchDaemons"
	launchAgentDir  = "Library/LaunchAgents" // relative to a user's home directory
)

// runCmd executes an external command and returns its combined output. It is
// a package-level variable so tests can substitute a mock without shelling
// out to a real launchctl.
var runCmd = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// darwinManager implements ServiceManager using launchd via launchctl.
type darwinManager struct{}

func label(name string) string { return fmt.Sprintf("com.serv.%s", name) }

// isUserLevel reports whether the service should run as a per-user
// LaunchAgent rather than a system-wide LaunchDaemon.
func isUserLevel(cfg *api.ServiceConfig) bool {
	return cfg.Account.Type == api.AccountTypeUser
}

func userAgentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, launchAgentDir), nil
}

// plistPath returns the plist path for a service, given whether it is
// user-level.
func plistPath(name string, userLevel bool) (string, error) {
	if userLevel {
		dir, err := userAgentDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, label(name)+".plist"), nil
	}
	return filepath.Join(launchDaemonDir, label(name)+".plist"), nil
}

// findPlistPath locates an existing plist for name, checking both the
// system LaunchDaemons and user LaunchAgents locations.
func findPlistPath(name string) (path string, userLevel bool, err error) {
	systemPath := filepath.Join(launchDaemonDir, label(name)+".plist")
	if _, statErr := os.Stat(systemPath); statErr == nil {
		return systemPath, false, nil
	}

	userPath, uErr := plistPath(name, true)
	if uErr == nil {
		if _, statErr := os.Stat(userPath); statErr == nil {
			return userPath, true, nil
		}
	}

	return "", false, fmt.Errorf("service %q not found", name)
}

// resolveLabel maps a user-supplied service name to a launchd label. If name
// corresponds to a serv-managed job (one created via Install), the
// com.serv.-prefixed label is used; otherwise name is treated as the literal
// label of a pre-existing launchd job, allowing serv to query and control
// jobs it did not install.
func resolveLabel(name string) string {
	if _, _, err := findPlistPath(name); err == nil {
		return label(name)
	}
	return name
}

func launchctl(args ...string) (string, error) {
	out, err := runCmd("launchctl", args...)
	if err != nil {
		return out, fmt.Errorf("launchctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out))
	}
	return out, nil
}

// Install generates a launchd plist from cfg and loads it.
func (d *darwinManager) Install(cfg *api.ServiceConfig) error {
	if cfg == nil || cfg.Name == "" {
		return fmt.Errorf("service config must have a name")
	}

	userLevel := isUserLevel(cfg)
	path, err := plistPath(cfg.Name, userLevel)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("service %q already exists", cfg.Name)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating plist directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(renderPlist(cfg)), 0o644); err != nil {
		return fmt.Errorf("writing plist file: %w", err)
	}

	if _, err := launchctl("load", path); err != nil {
		return err
	}

	if err := writeDarwinServiceConfig(cfg); err != nil {
		return fmt.Errorf("writing config for service %q: %w", cfg.Name, err)
	}

	return nil
}

// Remove unloads and deletes the plist for name.
func (d *darwinManager) Remove(name string) error {
	path, _, err := findPlistPath(name)
	if err != nil {
		return err
	}

	// Best-effort: the service may already be unloaded.
	_, _ = launchctl("unload", path)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist file: %w", err)
	}

	configDir := filepath.Dir(config.DefaultConfigPath(name))
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("removing config for service %q: %w", name, err)
	}

	return nil
}

// Start starts a stopped service via launchctl.
func (d *darwinManager) Start(name string) error {
	_, err := launchctl("start", resolveLabel(name))
	return err
}

// Stop stops a running service via launchctl.
func (d *darwinManager) Stop(name string) error {
	_, err := launchctl("stop", resolveLabel(name))
	return err
}

// Restart stops and starts a service via launchctl.
func (d *darwinManager) Restart(name string) error {
	if err := d.Stop(name); err != nil {
		return err
	}
	return d.Start(name)
}

// Status queries launchd for the current state of a service.
func (d *darwinManager) Status(name string) (ServiceStatus, error) {
	out, err := launchctl("list", resolveLabel(name))
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("service %q not found: %w", name, err)
	}

	pid := parsePlistInt(out, "PID")
	exitCode := parsePlistInt(out, "LastExitStatus")

	state := "stopped"
	if pid > 0 {
		state = "running"
	}

	return ServiceStatus{
		State:    state,
		PID:      pid,
		ExitCode: exitCode,
	}, nil
}

// List returns information about all launchd jobs on the system, including
// ones not installed by serv.
func (d *darwinManager) List() ([]ServiceInfo, error) {
	out, err := launchctl("list")
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
		if len(fields) < 3 {
			continue
		}
		pidField, statusField, jobLabel := fields[0], fields[1], fields[2]
		if pidField == "PID" {
			continue
		}

		name := strings.TrimPrefix(jobLabel, "com.serv.")
		pid, _ := strconv.Atoi(pidField)

		state := "stopped"
		if pid > 0 {
			state = "running"
		}
		_ = statusField

		infos = append(infos, ServiceInfo{
			Name:  name,
			State: state,
			PID:   pid,
		})
	}

	return infos, nil
}

// UpdateConfig unloads the job, regenerates the plist, and reloads it.
func (d *darwinManager) UpdateConfig(name string, cfg *api.ServiceConfig) error {
	if cfg == nil {
		return fmt.Errorf("service config must not be nil")
	}

	path, _, err := findPlistPath(name)
	if err != nil {
		return err
	}

	_, _ = launchctl("unload", path)

	newUserLevel := isUserLevel(cfg)
	newPath, err := plistPath(cfg.Name, newUserLevel)
	if err != nil {
		return err
	}
	if newPath != path {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing old plist file: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("creating plist directory: %w", err)
	}
	if err := os.WriteFile(newPath, []byte(renderPlist(cfg)), 0o644); err != nil {
		return fmt.Errorf("writing plist file: %w", err)
	}

	if _, err := launchctl("load", newPath); err != nil {
		return err
	}

	if err := writeDarwinServiceConfig(cfg); err != nil {
		return fmt.Errorf("writing config for service %q: %w", name, err)
	}

	return nil
}

// parsePlistInt extracts an integer value for key from launchctl's textual
// plist-style output, e.g. `"PID" = 4242;`.
func parsePlistInt(out, key string) int {
	needle := fmt.Sprintf("%q = ", key)
	idx := strings.Index(out, needle)
	if idx < 0 {
		return 0
	}
	rest := out[idx+len(needle):]
	end := strings.IndexAny(rest, ";\n")
	if end < 0 {
		end = len(rest)
	}
	v, _ := strconv.Atoi(strings.TrimSpace(rest[:end]))
	return v
}

// renderPlist builds a launchd plist from cfg.
func renderPlist(cfg *api.ServiceConfig) string {
	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")

	fmt.Fprintf(&b, "\t<key>Label</key>\n\t<string>%s</string>\n", label(cfg.Name))

	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	fmt.Fprintf(&b, "\t\t<string>%s</string>\n", cfg.Executable)
	for _, arg := range cfg.Arguments {
		fmt.Fprintf(&b, "\t\t<string>%s</string>\n", arg)
	}
	b.WriteString("\t</array>\n")

	if cfg.WorkingDirectory != "" {
		fmt.Fprintf(&b, "\t<key>WorkingDirectory</key>\n\t<string>%s</string>\n", cfg.WorkingDirectory)
	}

	if len(cfg.Environment) > 0 {
		b.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
		for k, v := range cfg.Environment {
			fmt.Fprintf(&b, "\t\t<key>%s</key>\n\t\t<string>%s</string>\n", k, v)
		}
		b.WriteString("\t</dict>\n")
	}

	if cfg.Account.Type == api.AccountTypeUser && cfg.Account.Username != "" {
		fmt.Fprintf(&b, "\t<key>UserName</key>\n\t<string>%s</string>\n", cfg.Account.Username)
	}

	keepAlive := cfg.Restart.Enabled == nil || *cfg.Restart.Enabled
	fmt.Fprintf(&b, "\t<key>KeepAlive</key>\n\t<%s/>\n", boolTag(keepAlive))

	if cfg.Restart.ThrottleCap.Unwrap() > 0 {
		fmt.Fprintf(&b, "\t<key>ThrottleInterval</key>\n\t<integer>%d</integer>\n", int(cfg.Restart.ThrottleCap.Unwrap().Seconds()))
	}

	if cfg.Stdout != "" {
		fmt.Fprintf(&b, "\t<key>StandardOutPath</key>\n\t<string>%s</string>\n", cfg.Stdout)
	}
	if cfg.Stderr != "" {
		fmt.Fprintf(&b, "\t<key>StandardErrorPath</key>\n\t<string>%s</string>\n", cfg.Stderr)
	}

	b.WriteString("</dict>\n</plist>\n")

	return b.String()
}

func boolTag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// writeDarwinServiceConfig marshals cfg to YAML and writes it to the
// service's config directory, creating the directory if necessary. It is a
// variable so tests can substitute it and avoid touching real config paths.
var writeDarwinServiceConfig = func(cfg *api.ServiceConfig) error {
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
