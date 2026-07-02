package hooks

import (
	"fmt"
	"os"
	"strings"
)

// Event names a service lifecycle point a hook can run at.
type Event string

const (
	EventPreStart  Event = "pre-start"
	EventPostStart Event = "post-start"
	EventPreStop   Event = "pre-stop"
	EventPostExit  Event = "post-exit"
	EventRotate    Event = "rotate"
)

// Context carries the data used to populate a hook process's SERV_*
// environment variables.
type Context struct {
	ServiceName    string
	PID            int
	ExitCode       int
	RuntimeSeconds int
	Event          Event
	// Action is free-form context describing what triggered the event,
	// e.g. "restart" or "stop".
	Action string
	Exe    string
	Args   []string
}

// buildEnv returns the environment for a hook process: the current
// process's environment plus SERV_* variables describing the lifecycle
// event.
func buildEnv(ctx Context) []string {
	env := os.Environ()
	return append(env,
		"SERV_SERVICE_NAME="+ctx.ServiceName,
		fmt.Sprintf("SERV_PID=%d", ctx.PID),
		fmt.Sprintf("SERV_EXIT_CODE=%d", ctx.ExitCode),
		fmt.Sprintf("SERV_RUNTIME_SECONDS=%d", ctx.RuntimeSeconds),
		"SERV_EVENT="+string(ctx.Event),
		"SERV_ACTION="+ctx.Action,
		"SERV_EXE="+ctx.Exe,
		"SERV_ARGS="+strings.Join(ctx.Args, " "),
	)
}
