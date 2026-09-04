# serv
[![CI](https://github.com/TillmanBuildsTech/serv/actions/workflows/ci.yml/badge.svg)](https://github.com/TillmanBuildsTech/serv/actions/workflows/ci.yml)
[![Release](https://github.com/TillmanBuildsTech/serv/actions/workflows/release.yml/badge.svg)](https://github.com/TillmanBuildsTech/serv/actions/workflows/release.yml)
![serv](https://github.com/TillmanBuildsTech/serv/blob/main/.github/assets/header.png)

**NSSM stopped development in 2017. serv is its modern successor — cross-platform, CLI-first, one YAML config, no GUI.**

Install any long-running executable as a native Windows service (via the SCM), or a systemd/launchd unit on Linux/macOS — all from one YAML config that behaves identically on every platform. serv is actively maintained and adds modern supervision NSSM never had: graceful shutdown escalation, automatic restart with backoff, stdout/stderr capture with rotation, lifecycle hooks, and process-tree killing.

> ⭐ **serv saves you time? Give it a star.** Maintenance, bug fixes, and new features land on a regular cadence — stars tell us the effort is worth it.
>
> 🔄 **Actively maintained.** Unlike NSSM (dormant since 2017), serv ships regular releases. See the [CHANGELOG](CHANGELOG.md).

<!--
TODO(demo-gif): Embed the quick-start demo GIF here once it's recorded.
Expected: .github/assets/demo.gif — `serv install --exe ... --name myapp`,
`serv start myapp`, `serv status myapp` succeeding on a real system.
Blocked on human action (kanban task t_5f884a45); replace this comment with
![demo](https://github.com/TillmanBuildsTech/serv/blob/main/.github/assets/demo.gif)
when the file exists.
-->

## serv vs NSSM vs WinSW vs Servy

| | **serv** | NSSM | WinSW | Servy |
|---|---|---|---|---|
| **Cross-platform** | ✅ **Windows, Linux, macOS** | ❌ Windows only | ❌ Windows only | ❌ Windows only |
| Config | One YAML file, portable | GUI / registry / INI | XML | GUI / CLI / PowerShell |
| CLI-first | ✅ yes | Partial (GUI-centric) | Partial | CLI + GUI |
| Auto-restart | ✅ with exponential backoff | ✅ basic | ✅ basic | ✅ auto-recovery |
| Graceful shutdown | ✅ escalating (Ctrl+C → close → terminate / SIGTERM → SIGKILL) | Basic | Basic | Basic |
| Log rotation | ✅ size- and age-based, timestamped | ✅ | ✅ | ✅ |
| Lifecycle hooks | ✅ pre-start / post-start / pre-stop / post-exit | Some | Some | Some |
| Maintained | ✅ **actively** | ❌ dormant since 2017 | ✅ | ✅ |

serv is the only one that runs the same config on Windows, Linux, and macOS — check your service definition into version control and use it everywhere.

## Features

- **One config, three platforms** — the same `ServiceConfig` YAML installs a
  Windows service, a systemd unit, or a launchd job.
- **Graceful shutdown escalation** — Windows: console Ctrl+C → window close →
  thread quit → terminate. Linux/macOS: SIGTERM → SIGKILL. Configurable
  per-stage timeouts.
- **Process tree killing** — stops/restarts kill the whole descendant
  process tree, with PID-reuse protection.
- **Automatic restart with backoff** — exponential backoff on repeated
  failures, resetting after sustained uptime, interruptible by a stop
  request.
- **Per-exit-code actions** — restart, ignore, exit cleanly, or trigger
  platform-level crash recovery, based on the child's exit code.
- **stdout/stderr capture and rotation** — line-buffered capture to log
  files, size- and age-based rotation with timestamped rotated filenames,
  safe interleaving when stdout and stderr share a file.
- **Lifecycle hooks** — run a command at `pre-start` (can abort the start),
  `post-start`, `pre-stop`, or `post-exit`, with a timeout and full
  environment context.
- **Windows account management** — LocalSystem/LocalService/NetworkService,
  virtual service accounts, or a custom domain/local user account
  (automatically granted the "Log on as a service" right).

## Quick start

### Install

| Platform | Command |
|---|---|
| Windows | `winget install TillmanBuildsTech.serv` · `choco install serv` |
| macOS | `brew install serv` |
| Any (Node.js) | `npm install -g @tillmanbuildstech/serv` |

### Run

Install, start, and check your service in three commands:

```sh
serv install --exe /path/to/myapp --name myapp
serv start myapp
serv status myapp
```

```
Name:   myapp
State:  running
PID:    12345
Uptime: 1m30s
Exe:    /path/to/myapp
Config: /etc/serv/myapp/config.yaml
```

List all services on the system, and stop/remove one you manage with serv when you're done:

```sh
serv list
serv stop myapp
serv remove myapp
```

For anything beyond the basics — restart policy, log rotation, hooks,
account configuration — write a YAML config and install from it:

```sh
serv install --config myapp.yaml
```

Whichever way you install, serv writes the resulting config to a per-OS
system directory (e.g. `C:\ProgramData\serv\myapp\config.yaml` on Windows,
`/etc/serv/myapp/config.yaml` on Linux) — see
[where config lives](docs/configuration.md#where-config-lives) for the full
list and how to update it afterward.

See the [serv configuration reference](docs/configuration.md) for every field,
and [serv lifecycle hooks](docs/hooks.md) for the lifecycle hook system.

## Documentation

serv is a modern, cross-platform successor to NSSM — the docs below cover
installing, configuring, and hooking into any app as a native Windows,
Linux, or macOS service.

- [Install serv](docs/installation.md) — run any app as a Windows, Linux, or
  macOS service; binary download, package managers, building from source.
- [serv configuration reference](docs/configuration.md) — every YAML field,
  its type, default, and an example.
- [serv lifecycle hooks](docs/hooks.md) — lifecycle events, environment
  variables, timeout behavior.
- [Release process](docs/release-process.md) — branch flow, versioning, and
  how pre-releases get built and published.
- [CHANGELOG](CHANGELOG.md)

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

Run the integration suite (some tests require Administrator/root and skip
themselves otherwise):

```bash
go test -tags=integration ./test/integration/...
```

### Running

```bash
./bin/serv
```

## License

[MIT](LICENSE)
