package cmd

import (
	"github.com/spf13/cobra"

	"github.com/TillmanBuildsTech/serv/internal/cli"
	"github.com/TillmanBuildsTech/serv/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "serv",
	Short: "A modern CLI tool",
	Long:  `A modern CLI tool built with Go and Cobra.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	cli.Register(rootCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("serv version " + version.Version)
	},
}
