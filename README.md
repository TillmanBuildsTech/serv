# Serv

Serv installs and supervises a long-running executable as a native Windows
service (via the SCM), or a systemd/launchd unit on Linux/macOS. It's a
cross-platform, modern alternative to tools like NSSM, adding process lifecycle
management, graceful shutdown, automatic restart with backoff, stdout/stderr
capture with rotation, and lifecycle hooks — all driven by one YAML config
that works the same way on every platform.

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

Install [Serv](docs/installation.md), then install and start a service:

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

See the [configuration reference](docs/configuration.md) for every field,
and [docs/hooks.md](docs/hooks.md) for the lifecycle hook system.

## Documentation

- [Installation](docs/installation.md) — binary download, package managers,
  building from source.
- [Configuration reference](docs/configuration.md) — every YAML field, its
  type, default, and an example.
- [Hooks](docs/hooks.md) — lifecycle events, environment variables, timeout
  behavior.
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
