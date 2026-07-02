// Package cli wires the serv command-line interface to the platform
// ServiceManager.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/platform"
)

// managerFactory constructs the ServiceManager used by CLI commands. It is
// a variable so tests can substitute platform.MockManager instead of
// hitting a real SCM/systemd/launchd.
var managerFactory = platform.NewServiceManager

// Register adds all service management subcommands to root.
func Register(root *cobra.Command) {
	root.AddCommand(
		newInstallCmd(),
		newRemoveCmd(),
		newStartCmd(),
		newStopCmd(),
		newRestartCmd(),
		newStatusCmd(),
		newListCmd(),
		newConfigCmd(),
	)
}
