package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/internal/process"
)

// processStartTime is a variable so tests can avoid depending on real OS
// process introspection.
var processStartTime = process.StartTime

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show the status of an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			status, err := managerFactory().Status(name)
			if err != nil {
				return fmt.Errorf("getting status for service %q: %w", name, err)
			}

			exe := "-"
			if cfg, err := config.Load(config.DefaultConfigPath(name)); err == nil {
				exe = cfg.Executable
			}

			uptime := "-"
			if status.PID > 0 {
				if start, ok := processStartTime(status.PID); ok {
					uptime = time.Since(start).Truncate(time.Second).String()
				}
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name:   %s\n", name)
			fmt.Fprintf(out, "State:  %s\n", status.State)
			fmt.Fprintf(out, "PID:    %s\n", pidString(status.PID))
			fmt.Fprintf(out, "Uptime: %s\n", uptime)
			fmt.Fprintf(out, "Exe:    %s\n", exe)
			return nil
		},
	}
}
