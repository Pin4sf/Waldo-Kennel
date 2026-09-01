# Kennel CLI

`kennel` is a thin Go/Cobra client for Kennel. It opens the desktop app and discovers, inspects, and stops the app-owned local daemon through loopback HTTP and the `running.json` handshake. It must not open SQLite directly or call runtime, workspace, tracker, or provider adapters in-process.

The source entrypoint remains `backend/cmd/kennel` as an audited AO synchronization seam. Local and packaged builds name the executable `kennel`; `ao` is not a supported alias for Kennel.

## Build from source

```sh
cd backend
go build -o ./bin/kennel ./cmd/kennel
./bin/kennel --help
```

Start the daemon before product commands, either through the desktop app or:

```sh
./bin/kennel start
./bin/kennel status --json
```

## Public command surface

The exact flags are authoritative in `kennel <command> --help`. The root help exposes lifecycle and support commands:

- `start`, `stop`, `status`, `doctor`, `dev`, `completion`, and `version`.

The desktop app is the product surface. Inherited runtime commands remain registered for operational compatibility, but are omitted from root help. The `daemon` and `start` lifecycle/bootstrap commands manage the local process or desktop-app boundary directly; `completion` and `version` render locally. The current `dev import-projects` subcommand validates local inputs before calling its daemon route. Every product command is a thin client to a daemon route. It must not open SQLite directly or call runtime, workspace, tracker, or provider adapters in-process. CLI misuse returns a usage error; daemon/runtime failures preserve the API error envelope and request ID where available.

## Session context

Commands resolve project/session context from explicit arguments first, then `KENNEL_PROJECT_ID` and `KENNEL_SESSION_ID`, then a registered project matching the current working directory. Pass an explicit project/session if inherited shell context is stale.

Examples:

```sh
kennel agent ls --refresh
kennel project ls
kennel spawn --project <project-id>
kennel session ls
kennel session get <session-id>
kennel session claim-pr <session-id> <pr-ref>
kennel send --session <session-id> --message "Inspect the failing check"
```

Provider switching, handoff, PR, review, preview, and browser subcommands retain the daemon-owned validation and capability checks. Run their local help before automation instead of copying an old AO invocation.

## Preview project formats

`kennel preview [target]` resolves the active session from `KENNEL_SESSION_ID`. An explicit URL opens directly; supported files can be served to the session Browser panel. `kennel preview start` may read the project-local `.kennel/launch.json` compatibility format. Agents must not create that file or install a framework/dev server unless the task calls for it.

`kennel browser` controls only the selected session's Electron browser surface. Its snapshots and references are session/tab scoped. Network metadata capture is off by default, bounded, excludes bodies and sensitive header values, and redacts URL credentials, fragments, and query values.

Project-local `.kennel/launch.json` and `.kennel/attachments` are inherited file-format names, not a path to AO global state.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `KENNEL_PORT` | `3031` | Primary loopback daemon port. |
| `KENNEL_RUN_FILE` | `~/.kennel/running.json` | PID/port handshake. |
| `KENNEL_DATA_DIR` | `~/.kennel/data` | SQLite and daemon data. |
| `KENNEL_REQUEST_TIMEOUT` | `60s` | REST request timeout. |
| `KENNEL_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown cap. |
| `KENNEL_KEEP_DAEMON` | unset / off | Keep the supervised daemon after the window closes. |
| `KENNEL_SESSION_ID` | unset | Active session context. |
| `KENNEL_PROJECT_ID` | unset | Active project context. |

The primary listener always binds `127.0.0.1`. A second listener can bind the Kennel LAN port only while Connect Mobile is explicitly enabled and bearer authentication is active; it cannot expose loopback-only control routes.

## Isolated smoke test

This test starts only the CLI-built daemon with temporary Kennel paths; it does not launch Electron. It uses the hidden `daemon` entrypoint directly because `kennel start` opens the desktop app:

```sh
cd backend
go build -o /tmp/kennel ./cmd/kennel

KENNEL_TEST_ROOT=$(mktemp -d)
export KENNEL_RUN_FILE="$KENNEL_TEST_ROOT/running.json"
export KENNEL_DATA_DIR="$KENNEL_TEST_ROOT/data"
export KENNEL_PORT=3037

/tmp/kennel daemon >"$KENNEL_TEST_ROOT/daemon.log" 2>&1 &
KENNEL_DAEMON_PID=$!
cleanup() {
  /tmp/kennel stop >/dev/null 2>&1 || true
  if kill -0 "$KENNEL_DAEMON_PID" 2>/dev/null; then
    kill "$KENNEL_DAEMON_PID" 2>/dev/null || true
  fi
  wait "$KENNEL_DAEMON_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

for _ in {1..100}; do
  /tmp/kennel status --json | grep -q '"state": "ready"' && break
  sleep 0.1
done
/tmp/kennel status --json | grep -q '"state": "ready"'
/tmp/kennel status --json
/tmp/kennel doctor
/tmp/kennel stop
wait "$KENNEL_DAEMON_PID"
trap - EXIT INT TERM
```

Remove the exact temporary directory only after verifying `KENNEL_TEST_ROOT` contains the path created for this smoke test.
