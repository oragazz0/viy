# Logging

> Structured JSON logging with zap.

## Overview

Viy uses [zap](https://github.com/uber-go/zap) for structured logging. All log output is JSON, written to stdout. Errors and internal zap failures go to stderr.

## Configuration

Set the log level with the `--log-level` flag:

```bash
viy unveil --eye disintegration --target deployment/nginx --log-level debug
```

| Level | Description |
|---|---|
| `debug` | Verbose output for development and troubleshooting |
| `info` | Default. Experiment lifecycle events |
| `warn` | Non-fatal issues (e.g., state persistence failures) |
| `error` | Operation failures with stack traces |

## Log Format

Every log entry is a JSON object with these fields:

| Field | Type | Description |
|---|---|---|
| `timestamp` | string | ISO 8601 timestamp |
| `level` | string | Lowercase level (`info`, `debug`, `warn`, `error`) |
| `message` | string | Human-readable event description |
| `caller` | string | Source file and line number (short path) |
| `stacktrace` | string | Stack trace (error level only) |

### Contextual Fields

The orchestrator and eyes add structured fields to log entries:

| Field | Added By | Description |
|---|---|---|
| `experiment_id` | Orchestrator | 12-character UUID prefix |
| `eye` | Orchestrator | Eye name (e.g., `disintegration`) |
| `target` | Orchestrator | Target resource name |
| `namespace` | Orchestrator | Kubernetes namespace |
| `dry_run` | Orchestrator | Whether dream mode is active |
| `total_pods` | Orchestrator | Number of pods resolved |
| `max_affected` | Orchestrator | Maximum pods allowed by blast radius |
| `blast_radius_pct` | Orchestrator | Configured blast radius percentage |
| `pod` | Disintegration Eye | Pod being deleted |
| `interval` | Disintegration Eye | Wait duration between kills |

## Example Output

```json
{"level":"info","timestamp":"2026-04-07T12:00:00.000Z","caller":"orchestrator/orchestrator.go:50","message":"experiment starting","experiment_id":"a1b2c3d4e5f6","eye":"disintegration","target":"nginx","namespace":"default","dry_run":false}
{"level":"info","timestamp":"2026-04-07T12:00:00.100Z","caller":"orchestrator/orchestrator.go:82","message":"targets resolved","total_pods":10,"max_affected":3,"blast_radius_pct":30}
{"level":"info","timestamp":"2026-04-07T12:00:00.200Z","caller":"disintegration/eye.go:100","message":"unveiling pod","pod":"nginx-abc123","namespace":"default"}
{"level":"info","timestamp":"2026-04-07T12:00:00.300Z","caller":"orchestrator/orchestrator.go:125","message":"revelation complete","experiment_id":"a1b2c3d4e5f6"}
```

## Filtering Logs

Since output is JSON, use `jq` to filter:

```bash
# Only errors
viy unveil ... 2>&1 | jq 'select(.level == "error")'

# Only a specific experiment
viy unveil ... 2>&1 | jq 'select(.experiment_id == "a1b2c3d4e5f6")'

# Pod-level events
viy unveil ... 2>&1 | jq 'select(.pod != null)'
```

## See Also

- [CLI Commands](../cli/commands.md) — `--log-level` flag
- [Architecture](../architecture/design.md) — how logging fits in the orchestrator flow
