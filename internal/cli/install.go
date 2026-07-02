package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func newInstallCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a new service",
	}

	f := bindServiceFlags(cmd)
	cmd.Flags().StringVar(&configPath, "config", "", "path to a YAML service config file (optional; flags alone are enough for basic usage)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var cfg *api.ServiceConfig
		if configPath != "" {
			loaded, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config %q: %w", configPath, err)
			}
			cfg = loaded
		} else {
			cfg = &api.ServiceConfig{}
			config.ApplyDefaults(cfg)
		}

		applyServiceFlags(cmd, cfg, f)

		if err := config.Validate(cfg); err != nil {
			return err
		}

		if err := managerFactory().Install(cfg); err != nil {
			return fmt.Errorf("installing service %q: %w", cfg.Name, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Service %q installed.\n", cfg.Name)
		return nil
	}

	return cmd
}
