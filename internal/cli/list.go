package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all services on the system",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			services, err := managerFactory().List()
			if err != nil {
				return fmt.Errorf("listing services: %w", err)
			}

			out := cmd.OutOrStdout()
			if len(services) == 0 {
				fmt.Fprintln(out, "No services found.")
				return nil
			}

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATE\tPID\tDISPLAY NAME")
			for _, s := range services {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.State, pidString(s.PID), s.DisplayName)
			}
			return w.Flush()
		},
	}
}
