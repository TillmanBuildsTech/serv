package api

type ServiceConfig struct {
	Name             string             `yaml:"name"`
	DisplayName      string             `yaml:"display_name"`
	Description      string             `yaml:"description"`
	Executable       string             `yaml:"executable"`
	Arguments        []string           `yaml:"arguments,omitempty"`
	WorkingDirectory string             `yaml:"working_directory,omitempty"`
	StartType        StartType          `yaml:"start_type"`
	StopMethod       StopConfig         `yaml:"stop_method"`
	Restart          RestartConfig      `yaml:"restart"`
	ExitActions      map[int]ExitAction `yaml:"exit_actions,omitempty"`
	Stdout           string             `yaml:"stdout,omitempty"`
	Stderr           string             `yaml:"stderr,omitempty"`
	Stdin            string             `yaml:"stdin,omitempty"`
	LogRotation      LogRotationConfig  `yaml:"log_rotation"`
	Account          AccountConfig      `yaml:"account,omitempty"`
	Environment      map[string]string  `yaml:"environment,omitempty"`
	KillProcessTree  *bool              `yaml:"kill_process_tree,omitempty"`
	Priority         string             `yaml:"priority,omitempty"`
	Affinity         string             `yaml:"affinity,omitempty"`
	Hooks            map[string]string  `yaml:"hooks,omitempty"`
	Dependencies     []string           `yaml:"dependencies,omitempty"`
	Recovery         RecoveryConfig     `yaml:"recovery,omitempty"`
}

type StartType string

const (
	StartTypeAuto    StartType = "auto"
	StartTypeManual  StartType = "manual"
	StartTypeDelayed StartType = "delayed"
)

type StopConfig struct {
	Methods          []StopMethod `yaml:"methods,omitempty"`
	ConsoleTimeout   Duration     `yaml:"console_timeout,omitempty"`
	WindowTimeout    Duration     `yaml:"window_timeout,omitempty"`
	ThreadsTimeout   Duration     `yaml:"threads_timeout,omitempty"`
	TerminateTimeout Duration     `yaml:"terminate_timeout,omitempty"`
}

type StopMethod string

const (
	StopMethodConsole   StopMethod = "console"
	StopMethodWindow    StopMethod = "window"
	StopMethodThreads   StopMethod = "threads"
	StopMethodTerminate StopMethod = "terminate"
)

type RestartConfig struct {
	Enabled     *bool    `yaml:"enabled,omitempty"`
	Delay       Duration `yaml:"delay,omitempty"`
	ThrottleCap Duration `yaml:"throttle_cap,omitempty"`
}

type ExitAction string

const (
	ExitActionRestart ExitAction = "restart"
	ExitActionIgnore  ExitAction = "ignore"
	ExitActionExit    ExitAction = "exit"
	ExitActionCrash   ExitAction = "crash"
)

type LogRotationConfig struct {
	Enabled        bool     `yaml:"enabled"`
	MaxBytes       int64    `yaml:"max_bytes,omitempty"`
	MaxAge         Duration `yaml:"max_age,omitempty"`
	OnlineRotation bool     `yaml:"online_rotation,omitempty"`
	// MinInterval is the minimum time that must pass between rotations,
	// preventing rapid successive rotations.
	MinInterval Duration `yaml:"min_interval,omitempty"`
	// TimestampLines prepends each log line with a "[2006-01-02
	// 15:04:05.000] " timestamp.
	TimestampLines bool `yaml:"timestamp_lines,omitempty"`
}

type AccountConfig struct {
	Username string      `yaml:"username,omitempty"`
	Password string      `yaml:"password,omitempty"`
	Type     AccountType `yaml:"type,omitempty"`
}

type AccountType string

const (
	AccountTypeLocalSystem    AccountType = "local_system"
	AccountTypeLocalService   AccountType = "local_service"
	AccountTypeNetworkService AccountType = "network_service"
	AccountTypeUser           AccountType = "user"
)

func BoolPtr(b bool) *bool { return &b }

// RecoveryConfig configures the Windows SCM's native failure-recovery
// actions for the serv service process itself, distinct from Restart
// (which governs restarting the supervised child process).
type RecoveryConfig struct {
	// Enabled sets fFailureActionsOnNonCrashFailures, so recovery triggers
	// on any non-zero exit of the service process, not just crashes.
	Enabled          bool           `yaml:"enabled"`
	FirstAction      RecoveryAction `yaml:"first_action,omitempty"`
	SecondAction     RecoveryAction `yaml:"second_action,omitempty"`
	SubsequentAction RecoveryAction `yaml:"subsequent_action,omitempty"`
	// RestartDelay is the delay before the SCM restarts the service after
	// a restart action.
	RestartDelay Duration `yaml:"restart_delay,omitempty"`
	// ResetPeriod is how long the service must run without failing before
	// the failure count resets to zero.
	ResetPeriod Duration `yaml:"reset_period,omitempty"`
	// RunCommand is the command line executed for RecoveryActionRunCommand.
	RunCommand string `yaml:"run_command,omitempty"`
	// RebootMessage is broadcast before a RecoveryActionReboot action.
	RebootMessage string `yaml:"reboot_message,omitempty"`
}

type RecoveryAction string

const (
	RecoveryActionNone       RecoveryAction = "none"
	RecoveryActionRestart    RecoveryAction = "restart"
	RecoveryActionRunCommand RecoveryAction = "run_command"
	RecoveryActionReboot     RecoveryAction = "reboot"
)
