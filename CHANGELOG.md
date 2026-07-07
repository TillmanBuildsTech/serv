# Changelog

All notable changes to this project are documented in this file.

## [0.1.7]

### Fixed

- `serv start`/`serv restart` no longer show a doubled error message (e.g.
  `starting service "appserver": starting service "appserver": ...`) when
  the underlying start fails; the CLI now passes the platform error through
  instead of re-wrapping it with the same prefix.
- On Windows, a start/restart that times out waiting for the service to
  report itself as running (`ERROR_SERVICE_REQUEST_TIMEOUT`) now says so
  explicitly and suggests checking the service's config file and logs,
  instead of only showing the generic Win32 message.
- `serv status` no longer silently hides the `Exe:`/`Config:` fields when a
  service's config file exists but fails to load (e.g. it was corrupted or
  left invalid YAML by an external edit). It now prints the config path
  along with the load error, so a broken config is visible instead of
  looking like the service has no config at all.

### Changed

- `docs/configuration.md` now includes `serv install`/`serv config`
  command-line examples alongside the existing `config.yaml` examples,
  including a flag-to-config-field mapping table.
- `docs/configuration.md` adds two more Node.js examples: running a
  TypeScript entry point via `ts-node`, and falling back to `npm.cmd` for
  `npm run <script>` when the script can't be resolved to a direct `node`
  invocation.

## [0.1.6]

### Added

- On Linux, `serv status` now also reports systemd's own native detail for a
  service (`Loaded`, `Since`, `Invocation`, `TriggeredBy`, `Docs`, `Tasks`,
  `Memory`, `CPU`, `CGroup`), the same information `systemctl status` shows,
  so managing a systemd-backed service through `serv` doesn't require also
  running `systemctl status` to see it. On macOS, `serv status` similarly
  reports the backing launchd plist path and label.

### Fixed

- `serv status` no longer prints an `Exe:` line for services that have no
  serv-authored config file, matching the existing `Config:` behavior. On
  Linux and macOS in particular, an `Exe: -` line for a service serv didn't
  install (e.g. `ssh`, managed natively by systemd/launchd) was noise that
  isn't part of what those native tools report.

## [0.1.5]

### Fixed

- `serv status` no longer prints a `Config:` path for services that have no
  serv-authored config file (e.g. services discovered on the system but not
  installed via `serv install`); the line is now omitted instead of pointing
  at a file that doesn't exist.

## [0.1.4]

### Fixed

- `list`, `status`, `start`, `stop`, and `restart` now consistently discover
  and control services on the whole system, not just ones `serv` itself
  installed, on all three platforms. Previously Windows leaked nearly every
  SCM service due to an overly broad `serv` substring filter (`"serv"` also
  matches the common word "service"), while Linux and macOS were scoped only
  to serv-managed units/jobs — the platforms disagreed and Linux/macOS `list`
  looked broken by comparison. `install`/`remove`/`update-config` still only
  operate on services `serv` created.
- Windows `remove`/`update-config` now refuse to touch a service that wasn't
  installed by serv, replacing the removed (and unreliable) list filter with
  an explicit safety check now that `list` surfaces every SCM service.

## [0.1.3]

### Changed

- Improved the release workflow's npm publishing step.

## [0.1.2]

### Added

- npm wrapper package (`packaging/npm/serv/`) so `npx @tillmanbuildstech/serv`
  or `npm install -g @tillmanbuildstech/serv` can install/run serv on any
  platform with Node.js — a postinstall script downloads and SHA256-verifies
  the matching release archive. Wired into the release pipeline's version
  bump alongside Homebrew/Scoop/winget.

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
