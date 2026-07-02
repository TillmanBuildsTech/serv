package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <name>",
		Short: "Update the configuration of an installed service",
		Args:  cobra.ExactArgs(1),
	}

	f := bindServiceFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load(config.DefaultConfigPath(name))
		if err != nil {
			return fmt.Errorf("loading existing config for service %q: %w", name, err)
		}

		applyServiceFlags(cmd, cfg, f)

		if err := config.Validate(cfg); err != nil {
			return err
		}

		if err := managerFactory().UpdateConfig(name, cfg); err != nil {
			return fmt.Errorf("updating service %q: %w", name, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Service %q updated.\n", name)
		return nil
	}

	return cmd
}
