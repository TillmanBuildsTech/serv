//go:build windows

package service

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"

	"github.com/TillmanBuildsTech/serv/internal/config"
	"github.com/TillmanBuildsTech/serv/pkg/api"
)

const acceptedControls = svc.AcceptStop | svc.AcceptShutdown

// Run loads the named service's configuration and starts the Windows SCM
// service runtime. It blocks until the SCM stops the service. Call it only
// when actually running under the SCM (i.e. from the "serv run <name>"
// entrypoint installed as the service's ImagePath).
func Run(name string) error {
	cfg, err := config.Load(config.DefaultConfigPath(name))
	if err != nil {
		return fmt.Errorf("loading config for service %q: %w", name, err)
	}
	return svc.Run(name, &handler{name: name, cfg: cfg})
}

// handler implements svc.Handler, registering the control handler via
// svc.Run and driving runtimeState through the service's lifecycle.
type handler struct {
	name string
	cfg  *api.ServiceConfig
}

// Execute is called by the SCM dispatcher once the service control handler
// is registered. It reports SERVICE_START_PENDING, starts the child
// process, reports SERVICE_RUNNING, then processes SCM control requests
// (stop/shutdown/interrogate) and child-process exits until the service
// stops.
func (h *handler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	s <- svc.Status{State: svc.StartPending}

	rt := newRuntimeState(h.name, h.cfg)

	if err := rt.startChild(); err != nil {
		s <- svc.Status{State: svc.Stopped}
		return true, 1
	}

	s <- svc.Status{State: svc.Running, Accepts: acceptedControls}

	for {
		select {
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				s <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				rt.stopChild()
				s <- svc.Status{State: svc.Stopped}
				return false, 0
			}

		case <-rt.done():
			switch rt.handleExit() {
			case actionExit:
				s <- svc.Status{State: svc.Stopped}
				return false, 0

			case actionCrash:
				s <- svc.Status{State: svc.Stopped}
				return true, 1

			case actionIgnore:
				// No active child; keep reporting Running and only react
				// to a future stop request.

			case actionRestart:
				delay := rt.nextBackoff()
				s <- svc.Status{State: svc.Paused, Accepts: acceptedControls}

				if stopRequested := waitBackoff(r, s, delay); stopRequested {
					rt.mp = nil // nothing to shut down; backoff already interrupted it
					s <- svc.Status{State: svc.Stopped}
					return false, 0
				}

				if err := rt.startChild(); err != nil {
					s <- svc.Status{State: svc.Stopped}
					return true, 1
				}
				s <- svc.Status{State: svc.Running, Accepts: acceptedControls}
			}
		}
	}
}

// waitBackoff blocks for delay while still responding to Interrogate
// requests, returning true immediately if a Stop/Shutdown request arrives
// during the wait — this is what makes restart backoff interruptible.
func waitBackoff(r <-chan svc.ChangeRequest, s chan<- svc.Status, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				s <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				return true
			}
		case <-timer.C:
			return false
		}
	}
}
