//go:build linux || darwin

package process

import (
	"syscall"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

const (
	defaultSigtermTimeout = 5 * time.Second
	defaultSigkillTimeout = 2 * time.Second
)

// buildShutdownStages returns the two-stage Unix shutdown escalation:
// SIGTERM to the process group, then SIGKILL to the process group.
func buildShutdownStages(pid int, cfg api.StopConfig) []shutdownStage {
	pgid := -pid // negative PID targets the process group (see setSysProcAttr's Setpgid).

	return []shutdownStage{
		{
			name:    "sigterm",
			timeout: durationOrDefault(cfg.TerminateTimeout, defaultSigtermTimeout),
			run: func() error {
				return syscall.Kill(pgid, syscall.SIGTERM)
			},
		},
		{
			name:    "sigkill",
			timeout: defaultSigkillTimeout,
			run: func() error {
				return syscall.Kill(pgid, syscall.SIGKILL)
			},
		},
	}
}
