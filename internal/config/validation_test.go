package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

func validConfig(t *testing.T) *api.ServiceConfig {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "app.exe")
	os.WriteFile(exe, []byte{}, 0755)

	return &api.ServiceConfig{
		Name:            "test",
		Executable:      exe,
		StartType:       api.StartTypeAuto,
		KillProcessTree: api.BoolPtr(true),
		Restart:         api.RestartConfig{Enabled: api.BoolPtr(true)},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig(t)
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	cfg := validConfig(t)
	cfg.Name = ""
	err := Validate(cfg)
	assertValidationError(t, err, "name")
}

func TestValidate_MissingExecutable(t *testing.T) {
	cfg := validConfig(t)
	cfg.Executable = ""
	err := Validate(cfg)
	assertValidationError(t, err, "executable")
}

func TestValidate_ExecutableNotFound(t *testing.T) {
	cfg := validConfig(t)
	cfg.Executable = "/nonexistent/binary"
	err := Validate(cfg)
	assertValidationError(t, err, "executable")
}

func TestValidate_InvalidStartType(t *testing.T) {
	cfg := validConfig(t)
	cfg.StartType = "invalid"
	err := Validate(cfg)
	assertValidationError(t, err, "start_type")
}

func TestValidate_NegativeTimeouts(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*api.ServiceConfig)
		field string
	}{
		{"console_timeout", func(c *api.ServiceConfig) { c.StopMethod.ConsoleTimeout = api.Duration(-1 * time.Second) }, "stop_method.console_timeout"},
		{"window_timeout", func(c *api.ServiceConfig) { c.StopMethod.WindowTimeout = api.Duration(-1 * time.Second) }, "stop_method.window_timeout"},
		{"threads_timeout", func(c *api.ServiceConfig) { c.StopMethod.ThreadsTimeout = api.Duration(-1 * time.Second) }, "stop_method.threads_timeout"},
		{"terminate_timeout", func(c *api.ServiceConfig) { c.StopMethod.TerminateTimeout = api.Duration(-1 * time.Second) }, "stop_method.terminate_timeout"},
		{"restart.delay", func(c *api.ServiceConfig) { c.Restart.Delay = api.Duration(-1 * time.Second) }, "restart.delay"},
		{"restart.throttle_cap", func(c *api.ServiceConfig) { c.Restart.ThrottleCap = api.Duration(-1 * time.Second) }, "restart.throttle_cap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(t)
			tt.setup(cfg)
			err := Validate(cfg)
			assertValidationError(t, err, tt.field)
		})
	}
}

func TestValidate_InvalidStopMethod(t *testing.T) {
	cfg := validConfig(t)
	cfg.StopMethod.Methods = []api.StopMethod{"invalid"}
	err := Validate(cfg)
	assertValidationError(t, err, "stop_method.methods")
}

func TestValidate_InvalidExitAction(t *testing.T) {
	cfg := validConfig(t)
	cfg.ExitActions = map[int]api.ExitAction{0: "invalid"}
	err := Validate(cfg)
	assertValidationError(t, err, "exit_actions[0]")
}

func TestValidate_NegativeLogRotation(t *testing.T) {
	cfg := validConfig(t)
	cfg.LogRotation.Enabled = true
	cfg.LogRotation.MaxBytes = -1
	err := Validate(cfg)
	assertValidationError(t, err, "log_rotation.max_bytes")
}

func TestValidate_UserAccountMissingUsername(t *testing.T) {
	cfg := validConfig(t)
	cfg.Account.Type = api.AccountTypeUser
	cfg.Account.Username = ""
	err := Validate(cfg)
	assertValidationError(t, err, "account.username")
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &api.ServiceConfig{}
	err := Validate(cfg)
	var verrs ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(verrs) < 2 {
		t.Errorf("expected multiple errors, got %d", len(verrs))
	}
}

func TestValidationErrors_ErrorString(t *testing.T) {
	errs := ValidationErrors{
		{Field: "name", Message: "is required"},
		{Field: "executable", Message: "is required"},
	}
	s := errs.Error()
	if s == "" {
		t.Fatal("expected non-empty error string")
	}
}

func assertValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error for field %q", field)
	}
	var verrs ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	for _, e := range verrs {
		if e.Field == field {
			return
		}
	}
	t.Errorf("expected error for field %q, got errors: %v", field, verrs)
}
