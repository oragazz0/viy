# State Persistence

> How Viy tracks experiments locally.

## Overview

Viy persists experiment metadata to a local JSON file at `~/.viy/state.json`. This enables the `vision` and `slumber` commands to inspect and manage experiments across CLI invocations.

## State File Location

```
~/.viy/state.json
```

- The `~/.viy/` directory is created automatically with `0700` permissions (owner only)
- The state file is written with `0600` permissions (owner read/write only)
- Writes are atomic: data is written to a `.tmp` file first, then renamed

## Experiment Schema

Each experiment in the state file has this structure:

```json
{
  "id": "a1b2c3d4e5f6",
  "status": "revealed",
  "eyes": ["disintegration"],
  "target": "nginx",
  "namespace": "default",
  "startTime": "2026-04-07T12:00:00Z",
  "endTime": "2026-04-07T12:05:00Z",
  "duration": 300000000000
}
```

| Field | Type | Description |
|---|---|---|
| `id` | string | 12-character UUID prefix, generated per experiment |
| `status` | string | Lifecycle state (see below) |
| `eyes` | []string | Names of eyes used |
| `target` | string | Target resource name |
| `namespace` | string | Kubernetes namespace |
| `startTime` | RFC 3339 | When the experiment started |
| `endTime` | RFC 3339 | When it ended (null if still running) |
| `duration` | int64 | Configured duration in nanoseconds |

## Experiment Lifecycle

```
unveiling ──► revealed    (success)
    │
    └──────► failed       (error during execution)

unveiling ──► revealed    (stopped via slumber)
```

| Status | Meaning |
|---|---|
| `unveiling` | Experiment is currently running |
| `revealed` | Experiment completed successfully or was stopped |
| `failed` | Experiment encountered an error |
| `paused` | Reserved for future use (pause/resume support) |

## State Transitions

The orchestrator manages state transitions:

1. **Start** — experiment is created with `status: unveiling`
2. **Success** — after `Unveil()` returns nil, status changes to `revealed`
3. **Failure** — if `Unveil()` returns an error, status changes to `failed`
4. **Slumber** — `viy slumber` sets active experiments to `revealed` in the state file

## Limitations

- **No cross-process coordination.** The state file records what happened, but `slumber` cannot cancel a running `unveil` process. The running process must be signaled directly (Ctrl+C or `kill`).
- **No automatic cleanup.** Old experiments accumulate in the state file. There is no TTL or garbage collection yet.
- **Single-machine scope.** The state file is local — experiments run on other machines are not visible.

## Example State File

```json
[
  {
    "id": "a1b2c3d4e5f6",
    "status": "revealed",
    "eyes": ["disintegration"],
    "target": "nginx",
    "namespace": "default",
    "startTime": "2026-04-07T12:00:00Z",
    "endTime": "2026-04-07T12:05:00Z",
    "duration": 300000000000
  },
  {
    "id": "c3d4e5f6a1b2",
    "status": "failed",
    "eyes": ["disintegration"],
    "target": "api",
    "namespace": "staging",
    "startTime": "2026-04-07T13:00:00Z",
    "endTime": "2026-04-07T13:00:01Z",
    "duration": 600000000000
  }
]
```

## See Also

- [CLI Commands](../cli/commands.md) — `vision` and `slumber` commands
- [Architecture](design.md) — where state fits in the overall design
- [Troubleshooting](../runbooks/troubleshooting.md) — state-related issues
