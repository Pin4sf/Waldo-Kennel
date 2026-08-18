# kennel stop

Stop the Kennel daemon.

## Syntax

```
kennel stop [flags]
```

## Flags

| Flag | Meaning | Default / Required |
|---|---|---|
| `--json` | Output stop result as JSON | - |
| `--timeout duration` | How long to wait for daemon shutdown | `10s` |

## Examples

```bash
# Stop the daemon
kennel stop
```

```bash
# Stop with a longer timeout
kennel stop --timeout 30s
```
