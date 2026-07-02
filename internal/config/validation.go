package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(msgs, "; "))
}

func Validate(cfg *api.ServiceConfig) error {
	var errs ValidationErrors

	if cfg.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "is required"})
	}

	if cfg.Executable == "" {
		errs = append(errs, ValidationError{Field: "executable", Message: "is required"})
	} else if _, err := os.Stat(cfg.Executable); err != nil {
		errs = append(errs, ValidationError{Field: "executable", Message: fmt.Sprintf("path does not exist: %s", cfg.Executable)})
	}

	switch cfg.StartType {
	case api.StartTypeAuto, api.StartTypeManual, api.StartTypeDelayed, "":
	default:
		errs = append(errs, ValidationError{Field: "start_type", Message: fmt.Sprintf("invalid value %q (must be auto, manual, or delayed)", cfg.StartType)})
	}

	if cfg.StopMethod.ConsoleTimeout < 0 {
		errs = append(errs, ValidationError{Field: "stop_method.console_timeout", Message: "must not be negative"})
	}
	if cfg.StopMethod.WindowTimeout < 0 {
		errs = append(errs, ValidationError{Field: "stop_method.window_timeout", Message: "must not be negative"})
	}
	if cfg.StopMethod.ThreadsTimeout < 0 {
		errs = append(errs, ValidationError{Field: "stop_method.threads_timeout", Message: "must not be negative"})
	}
	if cfg.StopMethod.TerminateTimeout < 0 {
		errs = append(errs, ValidationError{Field: "stop_method.terminate_timeout", Message: "must not be negative"})
	}
	for _, m := range cfg.StopMethod.Methods {
		switch m {
		case api.StopMethodConsole, api.StopMethodWindow, api.StopMethodThreads, api.StopMethodTerminate:
		default:
			errs = append(errs, ValidationError{Field: "stop_method.methods", Message: fmt.Sprintf("invalid method %q", m)})
		}
	}

	if cfg.Restart.Delay < 0 {
		errs = append(errs, ValidationError{Field: "restart.delay", Message: "must not be negative"})
	}
	if cfg.Restart.ThrottleCap < 0 {
		errs = append(errs, ValidationError{Field: "restart.throttle_cap", Message: "must not be negative"})
	}

	for code, action := range cfg.ExitActions {
		switch action {
		case api.ExitActionRestart, api.ExitActionIgnore, api.ExitActionExit, api.ExitActionCrash:
		default:
			errs = append(errs, ValidationError{Field: fmt.Sprintf("exit_actions[%d]", code), Message: fmt.Sprintf("invalid action %q", action)})
		}
	}

	if cfg.LogRotation.Enabled {
		if cfg.LogRotation.MaxBytes < 0 {
			errs = append(errs, ValidationError{Field: "log_rotation.max_bytes", Message: "must not be negative"})
		}
		if cfg.LogRotation.MaxAge < 0 {
			errs = append(errs, ValidationError{Field: "log_rotation.max_age", Message: "must not be negative"})
		}
	}

	if cfg.Account.Type == api.AccountTypeUser && cfg.Account.Username == "" {
		errs = append(errs, ValidationError{Field: "account.username", Message: "is required when account type is user"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
