package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/config"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.ValidateControlName(name); err != nil {
				return err
			}
			if err := managerFactory().Stop(name); err != nil {
				return fmt.Errorf("stopping service %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %q stopped.\n", name)
			return nil
		},
	}
}
