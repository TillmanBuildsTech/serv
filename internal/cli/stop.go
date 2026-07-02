package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := managerFactory().Stop(name); err != nil {
				return fmt.Errorf("stopping service %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %q stopped.\n", name)
			return nil
		},
	}
}
