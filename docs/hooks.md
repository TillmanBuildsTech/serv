# Lifecycle hooks

Hooks let you run an external command at specific points in a service's
lifecycle — for example, to warm up a cache before start, notify a
monitoring system, or clean up temp files after exit.

Hooks are configured under `hooks:` in the service YAML, mapping an event
name to a command line:

```yaml
hooks:
  pre-start: /opt/myapp/hooks/pre-start.sh
  post-start: /opt/myapp/hooks/post-start.sh
  pre-stop: /opt/myapp/hooks/pre-stop.sh
  post-exit: /opt/myapp/hooks/post-exit.sh
```

Each command is run through the platform shell (`cmd.exe /C` on Windows,
`/bin/sh -c` elsewhere), so you can use shell syntax — pipes, redirection,
multiple commands separated by `&&` — directly in the value.

## Events

| Event | When it fires | Effect of a non-zero exit or timeout |
|---|---|---|
| `pre-start` | Immediately before the child process is launched (on the initial start and on every restart). | **Aborts the start.** The child is not launched; serv retries according to the normal restart/backoff logic. |
| `post-start` | Immediately after the child process has been launched successfully. | Logged; does not affect the running child. |
| `pre-stop` | Before the graceful shutdown sequence begins (on an explicit stop, and internally before a SIGHUP-triggered reload on Linux/macOS). | Logged; shutdown proceeds regardless. |
| `post-exit` | After the child process has exited, for any reason (clean exit, crash, or being stopped). | Logged; does not affect anything further. |
| `rotate` | Reserved: intended to run before/after log rotation. Not currently invoked automatically. | — |

`pre-start` is the only event whose failure changes serv's behavior — it's
the mechanism for gating whether a service is allowed to start at all (e.g.
a readiness check, a license validation, or a "don't start during a
maintenance window" guard).

## Environment variables

Every hook process receives the current environment plus these variables
describing the event:

| Variable | Description |
|---|---|
| `SERV_SERVICE_NAME` | The service's name. |
| `SERV_PID` | The child process's PID. `0` if it hasn't started yet (e.g. during `pre-start`). |
| `SERV_EXIT_CODE` | The child's exit code. Only meaningful for `post-exit`. |
| `SERV_RUNTIME_SECONDS` | How long the child ran, in seconds. Only meaningful for `post-exit`. |
| `SERV_EVENT` | The event name, e.g. `pre-start`. |
| `SERV_ACTION` | Free-form context for the event (e.g. `restart` when a `pre-stop` fires as part of a reload). May be empty. |
| `SERV_EXE` | The configured `executable` path. |
| `SERV_ARGS` | The configured `arguments`, space-joined. |

## Timeout behavior

Each hook has a deadline, defaulting to **60 seconds**. If a hook doesn't
finish within its deadline, its entire process tree is killed (not just the
top-level process — the same tree-kill logic used for stopping the
supervised child), and the hook is treated as failed.

## Example: pre-start readiness gate

```yaml
hooks:
  pre-start: /opt/myapp/hooks/check-db.sh
```

```sh
#!/bin/sh
# check-db.sh: don't start the service until the database is reachable.
pg_isready -h db.internal -t 5
```

If `pg_isready` exits non-zero, the service start is aborted and serv
retries with the usual backoff, effectively waiting for the database to
become available before ever launching the child process.

## Example: post-exit notification

```yaml
hooks:
  post-exit: /opt/myapp/hooks/notify.sh
```

```sh
#!/bin/sh
# notify.sh: alert if the child exited with a non-zero code.
if [ "$SERV_EXIT_CODE" != "0" ]; then
  curl -s -X POST https://alerts.internal/notify \
    -d "service=$SERV_SERVICE_NAME exited with code $SERV_EXIT_CODE after ${SERV_RUNTIME_SECONDS}s"
fi
```
