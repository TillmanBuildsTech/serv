package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/config"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.ValidateName(name); err != nil {
				return err
			}
			if err := managerFactory().Remove(name); err != nil {
				return fmt.Errorf("removing service %q: %w", name, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service %q removed.\n", name)
			return nil
		},
	}
}
