# Configuration reference

Serv services are defined by a YAML file. You can generate one interactively
via `serv install --exe <path>` (which writes sensible defaults), or write
one by hand and pass it with `serv install --config service.yaml`.

Durations (anywhere a field's type is noted as `duration`) are Go duration
strings, e.g. `"1500ms"`, `"5s"`, `"2m"`, `"1h"`.

## Command-line flags

For simple services you don't need a `config.yaml` at all — `serv install`
accepts the same core fields as flags:

```sh
serv install --name myapp --exe C:\Apps\myapp\myapp.exe --workdir C:\Apps\myapp \
  --args "--port,8080" --display-name "My Application" --start-type auto
```

| Flag | Equivalent config field |
|---|---|
| `--name` | `name` |
| `--exe` | `executable` |
| `--args` | `arguments` (comma-separated, e.g. `--args "-u,main.py"`) |
| `--workdir` | `working_directory` |
| `--display-name` | `display_name` |
| `--description` | `description` |
| `--start-type` | `start_type` |
| `--config` | loads a full YAML file instead of/in addition to the flags above |

Flags and `--config` can be combined: the file is loaded first, then any
flags you explicitly pass override just those fields. Fields with no flag
equivalent (`stop_method`, `restart`, `exit_actions`, `log_rotation`,
`account`, `hooks`, etc.) can only be set via `--config`.

`serv config <name> [flags]` takes the same flags (minus `--config`) to
update an already-installed service without hand-editing its file:

```sh
serv config myapp --start-type manual --args "--port,9090"
```

Other lifecycle commands don't take service-definition flags, just the
service name:

```sh
serv start myapp
serv stop myapp
serv restart myapp
serv status myapp
serv remove myapp
serv list
```

## Where config lives

`serv install` writes a `config.yaml` for the service to a per-OS system
directory, keyed by the service's `name`:

| OS | Default path |
|---|---|
| Windows | `%PROGRAMDATA%\serv\<name>\config.yaml` (usually `C:\ProgramData\serv\<name>\config.yaml`) |
| macOS | `~/Library/Application Support/serv/<name>/config.yaml` |
| Linux | `/etc/serv/<name>/config.yaml` |

This is the file `serv status`, `serv config`, and friends read/update by
service name — you normally don't need to know the path at all.

To change settings after install, prefer `serv config <name> [flags]` (same
flags as `install`) rather than hand-editing the file, since it goes through
the platform's service manager (SCM/systemd/launchd) instead of just
rewriting YAML on disk. If you do want to inspect or hand-edit it directly,
the path above tells you where to look; on Linux/macOS you'll typically need
`sudo` to write there.

The base directory can be overridden with the `SERV_CONFIG_DIR` environment
variable (mainly useful for testing) — when set, config is written to
`$SERV_CONFIG_DIR/<name>/config.yaml` on every platform instead of the
default above.

## Top-level fields

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | *(required)* | Unique service identifier used by the SCM/systemd/launchd. If omitted on the `install` CLI, defaults to the executable's base name. |
| `display_name` | string | `""` | Human-readable name shown in `services.msc`, `systemctl status`, etc. |
| `description` | string | `""` | Longer description shown alongside the service. |
| `executable` | string | *(required)* | Path to the program serv launches and supervises. Must exist. |
| `arguments` | []string | `[]` | Arguments passed to the executable. |
| `working_directory` | string | executable's directory | Working directory for the child process. |
| `start_type` | string | `auto` | One of `auto`, `manual`, `delayed`. `delayed` is Windows-only (delayed auto-start); on Linux/macOS it is treated the same as `auto`. |
| `stop_method` | [StopConfig](#stopconfig) | see below | How the child is asked to shut down before being force-killed. |
| `restart` | [RestartConfig](#restartconfig) | see below | Backoff policy for restarting the child process after it exits. |
| `exit_actions` | map[int]string | `{}` | Maps a child exit code to an [exit action](#exit-actions). Codes not listed use the default action. |
| `stdout` | string | `""` (discarded) | File path the child's stdout is captured to. |
| `stderr` | string | `""` (discarded) | File path the child's stderr is captured to. If equal to `stdout`, both streams safely interleave into one file. |
| `stdin` | string | `""` (inherited/none) | File path opened and connected to the child's stdin. |
| `log_rotation` | [LogRotationConfig](#logrotationconfig) | see below | Rotation policy for `stdout`/`stderr` log files. |
| `account` | [AccountConfig](#accountconfig) | `local_system` | Which account the service runs as (Windows only; ignored on Linux/macOS, where the systemd/launchd unit's own user settings apply). |
| `environment` | map[string]string | `{}` | Extra environment variables set on the child process, in addition to the inherited environment. |
| `kill_process_tree` | bool | `true` | Whether stop/restart kills the child's entire descendant process tree, not just the immediate child. |
| `priority` | string | `normal` | Process priority class. Reserved for future use. |
| `affinity` | string | `""` | CPU affinity mask. Reserved for future use. |
| `hooks` | map[string]string | `{}` | Lifecycle event → shell command. See [hooks.md](hooks.md). |
| `dependencies` | []string | `[]` | Names of other services that must be running first. Reserved for future use. |
| `recovery` | [RecoveryConfig](#recoveryconfig) | disabled | Windows SCM native failure-recovery actions for the `serv` process itself (distinct from `restart`, which governs the supervised child). Windows only. |

## StopConfig

Controls the graceful shutdown escalation run before a service is forcefully
terminated. On Windows this is the NSSM-style four-stage sequence
(console Ctrl+C → window close → thread quit → terminate); on Linux/macOS
it's SIGTERM followed by SIGKILL, using `terminate_timeout` as the SIGTERM
wait and a short fixed wait before SIGKILL.

```yaml
stop_method:
  methods: [console, window, threads, terminate]
  console_timeout: 1500ms
  window_timeout: 1500ms
  threads_timeout: 1500ms
  terminate_timeout: 1500ms
```

| Field | Type | Default | Description |
|---|---|---|---|
| `methods` | []string | all four | Windows only. Subset of `console`, `window`, `threads`, `terminate` to attempt, in order. `terminate` always runs as a final fallback even if omitted. Ignored on Linux/macOS. |
| `console_timeout` | duration | `1500ms` | Windows only: time to wait after sending Ctrl+C before escalating. |
| `window_timeout` | duration | `1500ms` | Windows only: time to wait after posting `WM_CLOSE`. |
| `threads_timeout` | duration | `1500ms` | Windows only: time to wait after posting `WM_QUIT` to each thread. |
| `terminate_timeout` | duration | `1500ms` (Windows) | Windows: time to wait after `TerminateProcess` for the process to actually go away. Linux/macOS: how long to wait after SIGTERM before sending SIGKILL (default `5s` if unset). |

## RestartConfig

Governs restarting the supervised child process (not the `serv` process
itself — see [RecoveryConfig](#recoveryconfig) for that, on Windows).

```yaml
restart:
  enabled: true
  delay: 1s
  throttle_cap: 5m
```

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Whether the child is restarted at all after exiting (subject to `exit_actions`). |
| `delay` | duration | `1s` | Base backoff delay before the first restart attempt. |
| `throttle_cap` | duration | `5m` | Maximum backoff delay. Also the "sustained uptime" threshold: if the child ran at least this long before exiting, backoff resets to `delay` on the next failure. |

Backoff doubles on each consecutive failure (`delay`, `2×delay`, `4×delay`,
…) up to `throttle_cap`. The wait is interruptible: a stop request cancels
it immediately instead of waiting out the full delay.

## Exit actions

`exit_actions` maps a specific exit code to an action. Any exit code not
listed falls back to `restart` (or `exit` if `restart.enabled` is `false`).

```yaml
exit_actions:
  0: exit      # clean shutdown requested by the app itself — don't restart
  1: restart   # transient failure — restart with backoff
  2: ignore    # known "already running" code — leave it stopped, don't restart
  3: crash     # unrecoverable — report failure to the platform recovery mechanism
```

| Action | Behavior |
|---|---|
| `restart` | Restart the child with backoff (the default). |
| `ignore` | Leave the child stopped; the service itself stays reported as running/active, but nothing is supervised until a manual restart or a stop/start cycle. |
| `exit` | Stop supervising and report a clean stop (Windows: `SERVICE_STOPPED`; Linux/macOS: `serv run` exits 0, so systemd's `Restart=on-failure` won't restart it). |
| `crash` | Report failure. On Windows this triggers the SCM's [recovery](#recoveryconfig) actions if configured. On Linux/macOS, `serv run` exits non-zero so systemd's `Restart=on-failure` restarts the whole `serv` process. |

## LogRotationConfig

```yaml
log_rotation:
  enabled: true
  max_bytes: 10485760   # 10 MiB
  max_age: 168h         # 7 days
  online_rotation: true
  min_interval: 1m
  timestamp_lines: false
```

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Turns rotation on. When `false`, `stdout`/`stderr` are plain append-only files. |
| `max_bytes` | int | `10485760` (10 MiB) | Rotate once the active log file would exceed this size. Rotation always happens on a line boundary, never mid-line. |
| `max_age` | duration | `168h` (7 days) | Rotate once the active log file has been open at least this long. |
| `online_rotation` | bool | `false` | Reserved; rotation currently always happens live while the process runs (checked on every write), regardless of this flag. |
| `min_interval` | duration | none | Minimum time between rotations, to prevent rapid successive rotations under a very low `max_bytes`. |
| `timestamp_lines` | bool | `false` | Prepend each log line with a `[2006-01-02 15:04:05.000] ` timestamp. |

Rotated files are renamed to `<name>-<YYYYMMDDTHHMMSS>.log` next to the
active log file. If two rotations land in the same second, a numeric
suffix (`-1`, `-2`, …) is appended so an earlier rotated file is never
silently overwritten.

## AccountConfig

Windows only.

```yaml
account:
  type: user
  username: 'DOMAIN\svcuser'
  password: hunter2
```

| Field | Type | Default | Description |
|---|---|---|---|
| `type` | string | `local_system` | One of `local_system`, `local_service`, `network_service`, `user`. |
| `username` | string | — | Required when `type: user`. Either a `DOMAIN\user` account (requires `password`, and is automatically granted the "Log on as a service" right) or a virtual service account (`NT SERVICE\<ServiceName>`, no password needed). |
| `password` | string | — | Required when `type: user` and `username` is not a virtual service account. |

## RecoveryConfig

Windows only. Configures the SCM's own failure-recovery for the `serv`
process (visible in `services.msc`'s Recovery tab), separate from
`restart`, which governs the supervised child process.

```yaml
recovery:
  enabled: true
  first_action: restart
  second_action: restart
  subsequent_action: none
  restart_delay: 5s
  reset_period: 1h
```

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Sets `fFailureActionsOnNonCrashFailures`, so recovery triggers on any non-zero exit, not just crashes. |
| `first_action` | string | `none` | Action after the 1st failure: `none`, `restart`, `run_command`, or `reboot`. |
| `second_action` | string | `none` | Action after the 2nd failure. |
| `subsequent_action` | string | `none` | Action after the 3rd+ failure. |
| `restart_delay` | duration | `0` | Delay before the SCM restarts the service after a `restart` action. |
| `reset_period` | duration | `0` | How long the service must run without failing before the failure count resets. |
| `run_command` | string | `""` | Command line executed for a `run_command` action. |
| `reboot_message` | string | `""` | Message broadcast before a `reboot` action. |

## Examples: common executables

`executable` must be a concrete, launchable binary — not a script interpreter
invocation shorthand like `npm start`. For runtimes that aren't already
compiled native binaries, point `executable` at the runtime's binary and pass
your script/entry point via `arguments`.

**Go** (a compiled Go binary is just a native executable — nothing special
needed):

```yaml
executable: C:\Apps\myapp\myapp.exe   # or /usr/local/bin/myapp on Linux/macOS
arguments: []
working_directory: C:\Apps\myapp
```

```sh
serv install --name myapp --exe C:\Apps\myapp\myapp.exe --workdir C:\Apps\myapp
```

**Node.js / npm** — point at `node` directly rather than `npm`, so signals
and exit codes come from your app instead of being lost through npm's
wrapper process:

```yaml
executable: C:\Program Files\nodejs\node.exe   # or /usr/bin/node
arguments:
  - C:\Apps\myapp\server.js
working_directory: C:\Apps\myapp
environment:
  NODE_ENV: production
```

```sh
serv install --name myapp --exe "C:\Program Files\nodejs\node.exe" \
  --args "C:\Apps\myapp\server.js" --workdir C:\Apps\myapp
```

Note: `environment` has no CLI flag equivalent — set `NODE_ENV` etc. via
`--config` or by hand-editing the installed `config.yaml`.

**Node.js (TypeScript via `ts-node`)** — point at `node` and pass `ts-node`'s
CLI entry point from `node_modules`, rather than invoking the `ts-node`
shim script directly:

```yaml
executable: C:\Program Files\nodejs\node.exe   # or /usr/bin/node
arguments:
  - C:\Apps\myapp\node_modules\ts-node\dist\bin.js
  - C:\Apps\myapp\src\server.ts
working_directory: C:\Apps\myapp
environment:
  NODE_ENV: production
```

```sh
serv install --name myapp --exe "C:\Program Files\nodejs\node.exe" \
  --args "C:\Apps\myapp\node_modules\ts-node\dist\bin.js,C:\Apps\myapp\src\server.ts" \
  --workdir C:\Apps\myapp
```

For production it's usually better to run `tsc` as a build step and deploy
the compiled `dist/server.js`, pointing `executable`/`arguments` at that like
the plain Node.js example above — it starts faster and avoids shipping
`ts-node` and TypeScript itself to production.

**Node.js (`npm run <script>`)** — if what you actually need to run is a
script defined in `package.json` rather than a single entry file, `npm`
itself is still not a good fit for `executable` (see above), but `npm.cmd`
can be used as a last resort if you can't resolve the script to a direct
`node` invocation. On Windows, `npm.cmd` is itself a wrapper batch file, so
`kill_process_tree: true` (the default) matters here — otherwise stopping
the service leaves the real `node` child running:

```yaml
executable: C:\Program Files\nodejs\npm.cmd
arguments:
  - run
  - start
working_directory: C:\Apps\myapp
kill_process_tree: true
```

```sh
serv install --name myapp --exe "C:\Program Files\nodejs\npm.cmd" \
  --args "run,start" --workdir C:\Apps\myapp
```

**Python**:

```yaml
executable: C:\Apps\myapp\venv\Scripts\python.exe   # or /path/to/venv/bin/python3
arguments:
  - -u                        # unbuffered stdout/stderr, so logs show up promptly
  - C:\Apps\myapp\main.py
working_directory: C:\Apps\myapp
environment:
  PYTHONUNBUFFERED: "1"
```

```sh
serv install --name myapp --exe C:\Apps\myapp\venv\Scripts\python.exe \
  --args "-u,C:\Apps\myapp\main.py" --workdir C:\Apps\myapp
```

**Java (jar)**:

```yaml
executable: C:\Program Files\Java\jdk-21\bin\java.exe   # or /usr/bin/java
arguments:
  - -jar
  - C:\Apps\myapp\myapp.jar
  - --server.port=8080
working_directory: C:\Apps\myapp
```

```sh
serv install --name myapp --exe "C:\Program Files\Java\jdk-21\bin\java.exe" \
  --args "-jar,C:\Apps\myapp\myapp.jar,--server.port=8080" --workdir C:\Apps\myapp
```

**.NET** (self-contained executables work like Go; framework-dependent
builds need the `dotnet` host):

```yaml
executable: C:\Program Files\dotnet\dotnet.exe   # or /usr/bin/dotnet
arguments:
  - C:\Apps\myapp\myapp.dll
working_directory: C:\Apps\myapp
```

```sh
serv install --name myapp --exe "C:\Program Files\dotnet\dotnet.exe" \
  --args "C:\Apps\myapp\myapp.dll" --workdir C:\Apps\myapp
```

**Shell script** (Linux/macOS) — invoke the interpreter explicitly rather
than relying on the script's shebang and executable bit:

```yaml
executable: /bin/bash
arguments:
  - /opt/myapp/start.sh
working_directory: /opt/myapp
```

```sh
serv install --name myapp --exe /bin/bash --args "/opt/myapp/start.sh" --workdir /opt/myapp
```

Or with the POSIX `/bin/sh` if the script doesn't need bash-specific features:

```yaml
executable: /bin/sh
arguments:
  - /opt/myapp/start.sh
working_directory: /opt/myapp
```

```sh
serv install --name myapp --exe /bin/sh --args "/opt/myapp/start.sh" --workdir /opt/myapp
```

**Shell script (Windows)** — Windows has no `/bin/bash`. Point `executable`
at a bash you've installed instead, e.g. Git for Windows' bundled bash or
WSL's:

```yaml
executable: C:\Program Files\Git\bin\bash.exe   # Git for Windows
arguments:
  - .\ping.sh
working_directory: C:\Apps\myapp
```

```sh
serv install --name myapp --exe "C:\Program Files\Git\bin\bash.exe" \
  --args ".\ping.sh" --workdir C:\Apps\myapp
```

If you're using WSL, run the script through `wsl.exe` instead, since the
script and any paths it references need to resolve inside the Linux
filesystem:

```yaml
executable: C:\Windows\System32\wsl.exe
arguments:
  - bash
  - /opt/myapp/start.sh
working_directory: C:\Apps\myapp
```

```sh
serv install --name myapp --exe C:\Windows\System32\wsl.exe \
  --args "bash,/opt/myapp/start.sh" --workdir C:\Apps\myapp
```

Or, if the script doesn't rely on bash-specific syntax, skip bash entirely
and run it with PowerShell:

```yaml
executable: C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe
arguments:
  - -File
  - .\ping.ps1
working_directory: C:\Apps\myapp
```

```sh
serv install --name myapp --exe "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" \
  --args "-File,.\ping.ps1" --workdir C:\Apps\myapp
```

## Example: complete config

```yaml
name: myapp
display_name: My Application
description: An example service managed by serv

executable: C:\Apps\myapp\myapp.exe
arguments:
  - --port
  - "8080"
working_directory: C:\Apps\myapp

start_type: auto

stop_method:
  methods:
    - console
    - window
    - threads
    - terminate
  console_timeout: 1500ms
  window_timeout: 1500ms
  threads_timeout: 1500ms
  terminate_timeout: 1500ms

restart:
  enabled: true
  delay: 1s
  throttle_cap: 5m

exit_actions:
  0: exit
  1: restart
  2: crash

stdout: C:\Apps\myapp\logs\stdout.log
stderr: C:\Apps\myapp\logs\stderr.log

log_rotation:
  enabled: true
  max_bytes: 10485760  # 10 MB
  max_age: 168h        # 7 days
  online_rotation: true

account:
  type: local_system

environment:
  APP_ENV: production
  LOG_LEVEL: info

kill_process_tree: true
priority: normal

hooks:
  pre_start: C:\Apps\myapp\hooks\pre_start.bat
  post_stop: C:\Apps\myapp\hooks\post_stop.bat

dependencies:
  - postgresql
  - redis
```
