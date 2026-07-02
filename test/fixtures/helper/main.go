// Command helper is a standalone test fixture binary used by serv's
// integration test suite. It is not part of the serv CLI. It supports the
// scenarios integration tests need to exercise: long-running with periodic
// output, spawning a child process (for process-tree-kill tests), signal
// handling (including ignoring termination signals, to exercise shutdown
// escalation), and exiting with a configurable code either immediately or
// after a delay.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	outputInterval := flag.Duration("output-interval", 0, "write a line to stdout at this interval (0 disables periodic output)")
	exitAfter := flag.Duration("exit-after", 0, "exit automatically after this duration (0 = run until signaled)")
	exitCode := flag.Int("exit-code", 0, "exit code to use")
	spawnChild := flag.Bool("spawn-child", false, "spawn one child copy of this program (with -spawn-child=false) for process-tree-kill tests")
	ignoreTerm := flag.Bool("ignore-term", false, "ignore termination signals, to exercise shutdown escalation")
	pidFile := flag.String("pid-file", "", "if set, append this process's PID to the file (one per line), used to observe spawned children")
	flag.Parse()

	if *pidFile != "" {
		appendPID(*pidFile, os.Getpid())
	}

	if *spawnChild {
		args := []string{"-spawn-child=false"}
		if *pidFile != "" {
			args = append(args, "-pid-file", *pidFile)
		}
		if *ignoreTerm {
			args = append(args, "-ignore-term")
		}
		cmd := exec.Command(os.Args[0], args...)
		_ = cmd.Start()
	}

	sig := make(chan os.Signal, 1)
	if *ignoreTerm {
		signal.Ignore(syscall.SIGTERM, os.Interrupt)
	} else {
		signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	}

	var tick <-chan time.Time
	if *outputInterval > 0 {
		ticker := time.NewTicker(*outputInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	var deadline <-chan time.Time
	if *exitAfter > 0 {
		timer := time.NewTimer(*exitAfter)
		defer timer.Stop()
		deadline = timer.C
	}

	n := 0
	for {
		select {
		case <-sig:
			os.Exit(*exitCode)
		case <-deadline:
			os.Exit(*exitCode)
		case <-tick:
			n++
			fmt.Printf("tick %d pid=%d\n", n, os.Getpid())
		}
	}
}

func appendPID(path string, pid int) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%d\n", pid)
}
