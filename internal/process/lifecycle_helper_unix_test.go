//go:build linux || darwin

package process

import (
	"os/signal"
	"syscall"
)

// ignoreShutdownSignals makes the calling process immune to SIGTERM, used
// by the "ignore_signals" test helper to exercise shutdown escalation.
func ignoreShutdownSignals() {
	signal.Ignore(syscall.SIGTERM)
}
