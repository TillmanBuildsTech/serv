package main

import (
	"fmt"
	"os"

	"github.com/TillmanBuildsTech/serv/internal/cmd"
	"github.com/TillmanBuildsTech/serv/internal/service"
)

func main() {
	// The SCM/systemd/launchd launches the installed service as
	// "serv run <name>" (see internal/platform's Install implementations).
	// This bypasses the normal Cobra CLI entirely: service.Run blocks for
	// the life of the service and only returns once it has stopped.
	if len(os.Args) >= 3 && os.Args[1] == "run" {
		if err := service.Run(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
