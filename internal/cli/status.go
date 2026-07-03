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

			configPath := config.DefaultConfigPath(name)
			var exe string
			hasConfig := false
			if cfg, err := config.Load(configPath); err == nil {
				exe = cfg.Executable
				hasConfig = true
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
			if hasConfig {
				fmt.Fprintf(out, "Exe:    %s\n", exe)
				fmt.Fprintf(out, "Config: %s\n", configPath)
			}
			for _, d := range status.Detail {
				fmt.Fprintf(out, "%s %s\n", padLabel(d.Label), d.Value)
			}
			return nil
		},
	}
}

// detailLabelWidth is wide enough to fit the longest detail label ("TriggeredBy:")
// so platform-native detail fields line up in a column, like the fixed-width
// labels above them.
const detailLabelWidth = 12

// padLabel formats a detail field label as "Label:" padded to detailLabelWidth.
func padLabel(label string) string {
	return fmt.Sprintf("%-*s", detailLabelWidth, label+":")
}
