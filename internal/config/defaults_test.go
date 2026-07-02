package config

import (
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func TestApplyDefaults_EmptyConfig(t *testing.T) {
	cfg := &api.ServiceConfig{}
	ApplyDefaults(cfg)

	if cfg.StartType != api.StartTypeAuto {
		t.Errorf("expected start_type auto, got %s", cfg.StartType)
	}
	if len(cfg.StopMethod.Methods) != 4 {
		t.Errorf("expected 4 default stop methods, got %d", len(cfg.StopMethod.Methods))
	}
	if cfg.StopMethod.ConsoleTimeout != api.Duration(1500*time.Millisecond) {
		t.Errorf("expected console_timeout 1500ms, got %v", cfg.StopMethod.ConsoleTimeout)
	}
	if cfg.StopMethod.WindowTimeout != api.Duration(1500*time.Millisecond) {
		t.Errorf("expected window_timeout 1500ms, got %v", cfg.StopMethod.WindowTimeout)
	}
	if cfg.StopMethod.ThreadsTimeout != api.Duration(1500*time.Millisecond) {
		t.Errorf("expected threads_timeout 1500ms, got %v", cfg.StopMethod.ThreadsTimeout)
	}
	if cfg.StopMethod.TerminateTimeout != api.Duration(1500*time.Millisecond) {
		t.Errorf("expected terminate_timeout 1500ms, got %v", cfg.StopMethod.TerminateTimeout)
	}
	if cfg.Restart.Enabled == nil || !*cfg.Restart.Enabled {
		t.Error("expected restart.enabled defaulted to true")
	}
	if cfg.Restart.Delay != api.Duration(1*time.Second) {
		t.Errorf("expected restart.delay 1s, got %v", cfg.Restart.Delay)
	}
	if cfg.Restart.ThrottleCap != api.Duration(5*time.Minute) {
		t.Errorf("expected restart.throttle_cap 5m, got %v", cfg.Restart.ThrottleCap)
	}
	if cfg.KillProcessTree == nil || !*cfg.KillProcessTree {
		t.Error("expected kill_process_tree defaulted to true")
	}
	if cfg.LogRotation.MaxBytes != 10*1024*1024 {
		t.Errorf("expected max_bytes 10MB, got %d", cfg.LogRotation.MaxBytes)
	}
	if cfg.LogRotation.MaxAge != api.Duration(7*24*time.Hour) {
		t.Errorf("expected max_age 7d, got %v", cfg.LogRotation.MaxAge)
	}
	if cfg.Account.Type != api.AccountTypeLocalSystem {
		t.Errorf("expected account type local_system, got %s", cfg.Account.Type)
	}
	if cfg.Priority != "normal" {
		t.Errorf("expected priority normal, got %s", cfg.Priority)
	}
}

func TestApplyDefaults_PreservesExplicitValues(t *testing.T) {
	cfg := &api.ServiceConfig{
		StartType: api.StartTypeManual,
		StopMethod: api.StopConfig{
			ConsoleTimeout: api.Duration(5 * time.Second),
		},
		Restart: api.RestartConfig{
			Enabled: api.BoolPtr(false),
			Delay:   api.Duration(2 * time.Second),
		},
		KillProcessTree: api.BoolPtr(false),
		Priority:        "high",
	}
	ApplyDefaults(cfg)

	if cfg.StartType != api.StartTypeManual {
		t.Errorf("should preserve start_type manual, got %s", cfg.StartType)
	}
	if cfg.StopMethod.ConsoleTimeout != api.Duration(5*time.Second) {
		t.Errorf("should preserve console_timeout 5s, got %v", cfg.StopMethod.ConsoleTimeout)
	}
	if cfg.StopMethod.WindowTimeout != api.Duration(1500*time.Millisecond) {
		t.Errorf("should default window_timeout to 1500ms, got %v", cfg.StopMethod.WindowTimeout)
	}
	if *cfg.Restart.Enabled != false {
		t.Error("should preserve explicit restart.enabled=false")
	}
	if cfg.Restart.Delay != api.Duration(2*time.Second) {
		t.Errorf("should preserve restart.delay 2s, got %v", cfg.Restart.Delay)
	}
	if *cfg.KillProcessTree != false {
		t.Error("should preserve explicit kill_process_tree=false")
	}
	if cfg.Priority != "high" {
		t.Errorf("should preserve priority high, got %s", cfg.Priority)
	}
}
