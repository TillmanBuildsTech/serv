package cli

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// serviceFlags holds the CLI flags shared by commands that build or update
// a ServiceConfig (install, config).
type serviceFlags struct {
	name        string
	exe         string
	args        []string
	workdir     string
	displayName string
	description string
	startType   string
}

// bindServiceFlags registers the shared service flags on cmd.
func bindServiceFlags(cmd *cobra.Command) *serviceFlags {
	f := &serviceFlags{}
	cmd.Flags().StringVar(&f.name, "name", "", "service name (defaults to the executable's base name)")
	cmd.Flags().StringVar(&f.exe, "exe", "", "path to the executable to run as a service")
	cmd.Flags().StringSliceVar(&f.args, "args", nil, "arguments to pass to the executable")
	cmd.Flags().StringVar(&f.workdir, "workdir", "", "working directory for the executable")
	cmd.Flags().StringVar(&f.displayName, "display-name", "", "human-readable service name")
	cmd.Flags().StringVar(&f.description, "description", "", "service description")
	cmd.Flags().StringVar(&f.startType, "start-type", "", "start type: auto, manual, or delayed")
	return f
}

// applyServiceFlags overlays only the flags the user explicitly set onto
// cfg, so CLI flags override config file values without clobbering
// unspecified fields. If no name is set (by flag or config file) and an
// executable is available, the name defaults to the executable's base name
// with its extension stripped.
func applyServiceFlags(cmd *cobra.Command, cfg *api.ServiceConfig, f *serviceFlags) {
	if cmd.Flags().Changed("exe") {
		cfg.Executable = f.exe
	}
	if cmd.Flags().Changed("name") {
		cfg.Name = f.name
	} else if cfg.Name == "" && cfg.Executable != "" {
		cfg.Name = defaultNameFromExe(cfg.Executable)
	}
	if cmd.Flags().Changed("args") {
		cfg.Arguments = f.args
	}
	if cmd.Flags().Changed("workdir") {
		cfg.WorkingDirectory = f.workdir
	}
	if cmd.Flags().Changed("display-name") {
		cfg.DisplayName = f.displayName
	}
	if cmd.Flags().Changed("description") {
		cfg.Description = f.description
	}
	if cmd.Flags().Changed("start-type") {
		cfg.StartType = api.StartType(f.startType)
	}
}

func defaultNameFromExe(exe string) string {
	base := filepath.Base(exe)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// pidString formats a PID for display, showing "-" when the service isn't
// running (PID 0).
func pidString(pid int) string {
	if pid <= 0 {
		return "-"
	}
	return strconv.Itoa(pid)
}
