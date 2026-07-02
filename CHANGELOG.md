# Changelog

All notable changes to this project are documented in this file.

## [0.1.0]

Initial release.

### Added

- Cross-platform `ServiceManager` interface with Windows SCM, Linux systemd,
  and macOS launchd implementations (install/remove/start/stop/restart/
  status/list/update-config).
- Process launcher and monitor with exit-code and PID reporting.
- Graceful shutdown escalation: Windows console Ctrl+C → window close →
  thread quit → terminate; Linux/macOS SIGTERM → SIGKILL. Configurable
  per-stage timeouts and an interruptible-by-cancellation sequence.
- Process tree killing with PID-reuse protection via process start-time
  validation.
- Automatic restart with exponential backoff, resetting after sustained
  uptime, and interruptible backoff waits.
- Per-exit-code exit actions (`restart`, `ignore`, `exit`, `crash`).
- stdout/stderr capture to log files with line-buffered, interleave-safe
  writes when both streams share a file, and stdin redirection from a file.
- Log rotation by size and age, with timestamped rotated filenames,
  collision-safe naming, a configurable minimum interval between rotations,
  and optional per-line timestamps.
- Windows account management: well-known accounts, virtual service
  accounts, and custom accounts with automatic "Log on as a service" rights
  granting.
- Windows SCM native failure-recovery configuration, independent of the
  supervised child's own restart policy.
- Lifecycle hook executor (`pre-start`, `post-start`, `pre-stop`,
  `post-exit`) with environment context and a configurable timeout that
  kills a runaway hook's process tree.
- CLI: `install`, `remove`, `start`, `stop`, `restart`, `status`, `list`,
  `config`, wired to the platform `ServiceManager`.
- Windows SCM service runtime (`serv run <name>`) integrating process
  lifecycle, shutdown, restart backoff, I/O redirection, and hooks into the
  `StartServiceCtrlDispatcher` control loop.
- Linux/macOS foreground service runtime (`serv run <name>`) supervising the
  child process under systemd/launchd, with SIGTERM graceful shutdown,
  SIGHUP config reload, and systemd readiness notification.
- Integration test suite covering process tree killing, I/O capture, log
  rotation, restart backoff, hook abort/allow, and the full real
  install → start → stop → remove service lifecycle.
