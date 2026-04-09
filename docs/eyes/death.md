# Eye of Death

> Resource exhaustion — reveals truths about resource limits, HPA scaling, and OOM killer behavior.

## Overview

The Eye of Death injects controlled resource stress into pods via ephemeral containers running [stress-ng](https://github.com/ColinIanKing/stress-ng). It answers questions like:

- Are resource limits and requests configured correctly?
- Does the Horizontal Pod Autoscaler scale under pressure?
- How does the application behave under CPU, memory, or disk I/O contention?
- Does the OOM killer terminate the right processes?

## Mechanism

Death injects stress-ng as an **ephemeral container** into each target pod. This stresses the pod's own cgroup — not just the node — giving accurate pod-level resource pressure.

**Lifecycle:**

1. Discover target pods via label selector
2. For each pod, inject an ephemeral container running stress-ng with the configured parameters
3. stress-ng runs for the configured duration (with optional ramp-up)
4. On Pause or Close, the stress-ng process is killed via `kill 1` exec into the ephemeral container

**Partial failure handling:** If injection fails for some pods but succeeds for others, Unveil succeeds with a degraded result. The partial failure is surfaced as a truth in `Observe()` metrics.

> Note: Ephemeral containers are append-only in Kubernetes. The container entry remains on the pod spec after the process exits, but the process itself is fully terminated on Pause/Close.

## Configuration

Pass eye-specific config via `--config` as comma-separated `key=value` pairs.

| Key | Type | Default | Description |
|---|---|---|---|
| `cpuStress` | int (1-100) | `0` | Percentage of CPU load per worker |
| `memoryStress` | int (1-100) | `0` | Percentage of memory to consume per worker |
| `diskIOBytes` | int64 | `0` | Bytes per worker for disk I/O stress |
| `duration` | duration | — | How long stress-ng runs (required) |
| `rampUp` | duration | `0s` | Gradual ramp-up period before full stress |
| `workers` | int | — | Number of stress-ng worker threads (required) |

### Validation Rules

- At least one stress type must be enabled (`cpuStress`, `memoryStress`, or `diskIOBytes` > 0)
- Percentages must be between 1% and 100%
- `diskIOBytes` must be non-negative
- `duration` must be positive
- `rampUp` must be less than `duration`
- `workers` must be at least 1

### stress-ng Command Mapping

The config maps to stress-ng flags as follows:

| Config | stress-ng Flags |
|---|---|
| `cpuStress: 80, workers: 4` | `--cpu 4 --cpu-load 80` |
| `memoryStress: 70, workers: 4` | `--vm 4 --vm-bytes 70%` |
| `diskIOBytes: 1048576, workers: 4` | `--hdd 4 --hdd-bytes 1048576` |
| `rampUp: 30s` | `--ramp-time 30` |
| `duration: 2m` | `--timeout 120` |

## Examples

CPU stress only (80% load, 4 workers, 2 minutes):

```bash
viy unveil --eye death --target deployment/api-server \
  --config "cpuStress=80,workers=4,duration=2m"
```

All stressors with ramp-up:

```bash
viy unveil --eye death --target deployment/api-server \
  --config "cpuStress=80,memoryStress=70,diskIOBytes=1048576,duration=2m,rampUp=30s,workers=4"
```

Memory stress to test OOM behavior:

```bash
viy unveil --eye death --target deployment/worker \
  --config "memoryStress=95,workers=2,duration=1m"
```

Preview without executing:

```bash
viy dream --eye death --target deployment/api-server \
  --config "cpuStress=80,workers=4,duration=2m"
```

Combine with safety constraints:

```bash
viy unveil --eye death --target deployment/api-server \
  --config "cpuStress=80,workers=4,duration=2m" \
  --blast-radius 50% \
  --min-healthy 2
```

## Execution Flow

1. Resolve the target resource via the Kubernetes API
2. Extract the resource's pod selector, merge with any user-supplied `--selector`
3. List matching pods
4. Build stress-ng command from config (CPU, memory, disk I/O stressors + ramp-up + timeout)
5. For each pod:
   - Generate a unique ephemeral container name (`viy-death-<pod-prefix>`)
   - Inject ephemeral container with stress-ng command
   - If injection fails, log the error and continue to next pod
   - Track successfully injected containers for cleanup
6. Record truths: how many pods were affected, whether any failed

## Metrics

During and after execution, `Observe()` returns:

| Metric | Description |
|---|---|
| `EyeName` | Always `"death"` |
| `TargetsAffected` | Number of pods where stress was successfully injected |
| `OperationsTotal` | Total injection API calls made |
| `ErrorsTotal` | Injection operations that failed |
| `TruthsRevealed` | Summary of what was revealed, including partial failures |
| `IsActive` | Whether the eye is currently active |

## Error Handling

| Error | Cause | What to do |
|---|---|---|
| `target not found` | No pods match the selector | Check the target name, namespace, and selector |
| `invalid configuration` | Config validation failed | Check that at least one stress type is enabled and values are in range |
| `failed to discover targets` | Kubernetes API error | Check cluster connectivity and RBAC permissions |
| Partial failure (surfaced as truth) | Injection failed for some pods | Check pod status and ephemeral container support on the cluster |

## Prerequisites

- Kubernetes 1.25+ (ephemeral containers GA)
- The stress-ng container image (`alexeiled/stress-ng:latest`) must be pullable from target nodes
- RBAC permissions to patch pods (ephemeral containers subresource)

## See Also

- [Eyes Overview](overview.md) — interface and registry
- [Safety Guards](../safety/guards.md) — blast radius and min healthy constraints
- [CLI Commands](../cli/commands.md) — `unveil` and `dream` usage
