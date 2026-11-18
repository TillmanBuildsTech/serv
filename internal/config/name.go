package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// maxServiceNameLen matches the Windows SCM limit for service names.
const maxServiceNameLen = 256

// serviceNamePattern is the set of names serv will create or manage files
// for: a leading letter or digit, then letters, digits, '.', '_' or '-'.
// It excludes path separators and the "." / ".." path elements, so a name
// can never escape the config root when joined onto it.
var serviceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidateName checks a name for a service serv installs, updates, removes
// or runs (anything that maps the name onto files serv owns).
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid service name: must not be empty")
	}
	if len(name) > maxServiceNameLen {
		return fmt.Errorf("invalid service name %q: longer than %d characters", name, maxServiceNameLen)
	}
	if !serviceNamePattern.MatchString(name) {
		return fmt.Errorf("invalid service name %q: must start with a letter or digit and contain only letters, digits, '.', '_' or '-'", name)
	}
	return nil
}

// ValidateControlName checks a name passed to start, stop, restart or
// status. Those commands can also target services serv did not install
// (e.g. a systemd template instance like "getty@tty1" or a Windows service
// whose name contains spaces), so the rule is looser than ValidateName, but
// it still rejects anything that could act as a path: separators, "." and
// "..", control characters, and a leading '-' (which would be parsed as an
// option by systemctl/launchctl).
func ValidateControlName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("invalid service name: must not be empty")
	case len(name) > maxServiceNameLen:
		return fmt.Errorf("invalid service name %q: longer than %d characters", name, maxServiceNameLen)
	case name == "." || name == "..":
		return fmt.Errorf("invalid service name %q", name)
	case strings.HasPrefix(name, "-"):
		return fmt.Errorf("invalid service name %q: must not start with '-'", name)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("invalid service name %q: must not contain '/' or '\\'", name)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("invalid service name %q: must not contain control characters", name)
		}
	}
	return nil
}

// ServiceConfigDir returns <config root>/<name> after validating name and
// confirming the resolved directory is a direct child of the config root.
// Callers that delete a service's config directory must use this rather
// than deriving a directory from DefaultConfigPath.
func ServiceConfigDir(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	root, err := filepath.Abs(ConfigRoot())
	if err != nil {
		return "", fmt.Errorf("resolving config root: %w", err)
	}
	dir := filepath.Join(root, name)
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel != name || rel == "." || strings.HasPrefix(rel, "..") || strings.ContainsAny(rel, `/\`) {
		return "", fmt.Errorf("refusing to use config directory %q: not a direct child of %q", dir, root)
	}
	return dir, nil
}
