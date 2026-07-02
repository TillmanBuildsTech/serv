//go:build linux || darwin

package service

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TillmanBuildsTech/serv/internal/config"
)

// Run loads the named service's configuration and runs the foreground
// supervisor loop: it launches and monitors the child process until a stop
// signal is received or the child's resolved exit action ends supervision.
// systemd/launchd are responsible for restarting the serv process itself
// (via the unit's Restart= policy); Run only supervises the wrapped child.
func Run(name string) error {
	cfg, err := config.Load(config.DefaultConfigPath(name))
	if err != nil {
		return fmt.Errorf("loading config for service %q: %w", name, err)
	}

	rt := newRuntimeState(name, cfg)
	if err := rt.startChild(); err != nil {
		return fmt.Errorf("starting service %q: %w", name, err)
	}

	notifyReady()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer signal.Stop(sig)

	for {
		select {
		case s := <-sig:
			switch s {
			case syscall.SIGHUP:
				_ = rt.reload() // reload failure: keep supervising under the previous config
			case syscall.SIGTERM, syscall.SIGINT:
				rt.stopChild()
				return nil
			}

		case <-rt.done():
			switch rt.handleExit() {
			case actionExit:
				return nil
			case actionCrash:
				return fmt.Errorf("service %q exited requesting crash recovery", name)
			case actionIgnore:
				// No active child; loop back and only react to signals.
			case actionRestart:
				delay := rt.nextBackoff()
				if stopRequested := waitBackoffUnix(sig, delay); stopRequested {
					return nil
				}
				if err := rt.startChild(); err != nil {
					return fmt.Errorf("restarting service %q: %w", name, err)
				}
			}
		}
	}
}

// waitBackoffUnix blocks for delay while still responding to SIGHUP
// (ignored — there's no running child to reload), returning true
// immediately if SIGTERM/SIGINT arrives during the wait. This makes
// restart backoff interruptible.
func waitBackoffUnix(sig <-chan os.Signal, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case s := <-sig:
			switch s {
			case syscall.SIGTERM, syscall.SIGINT:
				return true
			}
		case <-timer.C:
			return false
		}
	}
}

// notifyReady sends "READY=1" to systemd via the sd_notify protocol if
// NOTIFY_SOCKET is set (i.e. the unit is configured with Type=notify). It
// is a no-op otherwise, including on macOS, which has no systemd.
func notifyReady() {
	socketPath := os.Getenv("NOTIFY_SOCKET")
	if socketPath == "" {
		return
	}

	addr := &net.UnixAddr{Name: socketPath, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	_, _ = conn.Write([]byte("READY=1"))
}
