package config

import (
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func ApplyDefaults(cfg *api.ServiceConfig) {
	if cfg.StartType == "" {
		cfg.StartType = api.StartTypeAuto
	}

	if cfg.StopMethod.Methods == nil {
		cfg.StopMethod.Methods = []api.StopMethod{
			api.StopMethodConsole,
			api.StopMethodWindow,
			api.StopMethodThreads,
			api.StopMethodTerminate,
		}
	}
	if cfg.StopMethod.ConsoleTimeout == 0 {
		cfg.StopMethod.ConsoleTimeout = api.Duration(1500 * time.Millisecond)
	}
	if cfg.StopMethod.WindowTimeout == 0 {
		cfg.StopMethod.WindowTimeout = api.Duration(1500 * time.Millisecond)
	}
	if cfg.StopMethod.ThreadsTimeout == 0 {
		cfg.StopMethod.ThreadsTimeout = api.Duration(1500 * time.Millisecond)
	}
	if cfg.StopMethod.TerminateTimeout == 0 {
		cfg.StopMethod.TerminateTimeout = api.Duration(1500 * time.Millisecond)
	}

	if cfg.Restart.Enabled == nil {
		cfg.Restart.Enabled = api.BoolPtr(true)
	}
	if cfg.Restart.Delay == 0 {
		cfg.Restart.Delay = api.Duration(1 * time.Second)
	}
	if cfg.Restart.ThrottleCap == 0 {
		cfg.Restart.ThrottleCap = api.Duration(5 * time.Minute)
	}

	if cfg.KillProcessTree == nil {
		cfg.KillProcessTree = api.BoolPtr(true)
	}

	if cfg.LogRotation.MaxBytes == 0 {
		cfg.LogRotation.MaxBytes = 10 * 1024 * 1024
	}
	if cfg.LogRotation.MaxAge == 0 {
		cfg.LogRotation.MaxAge = api.Duration(7 * 24 * time.Hour)
	}

	if cfg.Account.Type == "" {
		cfg.Account.Type = api.AccountTypeLocalSystem
	}

	if cfg.Priority == "" {
		cfg.Priority = "normal"
	}
}
