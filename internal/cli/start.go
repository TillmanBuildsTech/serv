package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := managerFactory().Start(name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %q started.\n", name)
			return nil
		},
	}
}
