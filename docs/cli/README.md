# Kennel CLI

`kennel` is a thin Go/Cobra client for the local Kennel daemon. It starts, discovers, inspects, and stops the daemon through loopback HTTP and the `running.json` handshake. It must not open SQLite directly or call runtime, workspace, tracker, or provider adapters in-process.

The source entrypoint remains `backend/cmd/ao` as an audited AO synchronization seam. Local and packaged builds name the executable `kennel`; `ao` is not a supported alias for Kennel.

## Build from source

```sh
cd backend
go build -o ./bin/kennel ./cmd/ao
./bin/kennel --help
```

Start the daemon before product commands, either through the desktop app or:

```sh
./bin/kennel start
./bin/kennel status --json
```

## Command families

The exact flags are authoritative in `kennel <command> --help`. Current command families include:

- daemon control: `start`, `stop`, `status`, `doctor`, `completion`, `version`;
- projects and agents: `project`, `agent`;
- workers and orchestrators: `spawn`, `session`, `orchestrator`, `send`;
- source control and reviews: `pr`, `review`;
- visual work: `preview`, `browser`;
- internal agent integration: `hooks` and the hidden `daemon` entrypoint.

Every product command resolves to a daemon route. CLI misuse returns a usage error; daemon/runtime failures preserve the API error envelope and request ID where available.

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

`kennel preview [target]` resolves the active session from `KENNEL_SESSION_ID`. An explicit URL opens directly; supported files can be served to the session Browser panel. `kennel preview start` may read the project-local `.ao/launch.json` compatibility format. Agents must not create that file or install a framework/dev server unless the task calls for it.

`kennel browser` controls only the selected session's Electron browser surface. Its snapshots and references are session/tab scoped. Network metadata capture is off by default, bounded, excludes bodies and sensitive header values, and redacts URL credentials, fragments, and query values.

Project-local `.ao/launch.json` and `.ao/attachments` are inherited file-format names, not a path to AO global state.

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

This test starts only the CLI-built daemon with temporary Kennel paths; it does not launch Electron:

```sh
cd backend
go build -o /tmp/kennel ./cmd/ao

KENNEL_TEST_ROOT=$(mktemp -d)
export KENNEL_RUN_FILE="$KENNEL_TEST_ROOT/running.json"
export KENNEL_DATA_DIR="$KENNEL_TEST_ROOT/data"
export KENNEL_PORT=3037

/tmp/kennel status --json
/tmp/kennel doctor
/tmp/kennel start
/tmp/kennel status --json
/tmp/kennel stop
```

Remove the exact temporary directory only after verifying `KENNEL_TEST_ROOT` contains the path created for this smoke test.
