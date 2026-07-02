package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := managerFactory().Restart(name); err != nil {
				return fmt.Errorf("restarting service %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %q restarted.\n", name)
			return nil
		},
	}
}
