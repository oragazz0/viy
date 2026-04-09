# Eye of Disintegration

> Pod termination — reveals auto-recovery and orchestration health.

## Overview

The Eye of Disintegration deletes pods to test whether your Kubernetes orchestration can recover. It answers questions like:

- Does my Deployment replace killed pods quickly enough?
- Does my application handle pod disruption gracefully?
- Are my readiness probes configured correctly?

## Configuration

Pass eye-specific config via `--config` as comma-separated `key=value` pairs.

| Key | Type | Default | Description |
|---|---|---|---|
| `podKillCount` | int | `1` | Exact number of pods to kill |
| `podKillPercentage` | int (1-100) | — | Percentage of matched pods to kill |
| `strategy` | string | `random` | Pod selection: `random` or `sequential` |
| `interval` | duration | `0s` | Wait time between consecutive pod kills |
| `gracePeriod` | duration | `0s` | Kubernetes graceful termination period |

### Kill Count Rules

- Specify **either** `podKillCount` **or** `podKillPercentage`, not both.
- When using `podKillPercentage`, the count is calculated as `total_pods * percentage / 100`, with a minimum of 1.
- If the requested kill count exceeds available pods, the experiment fails with `insufficient targets`.

### Selection Strategies

- **`random`** (default) — shuffles the pod list and picks the first N. Each run affects different pods.
- **`sequential`** — takes the first N pods in the order returned by the Kubernetes API. Deterministic across runs.

## Examples

Kill 1 random pod (defaults):

```bash
viy unveil --eye disintegration --target deployment/nginx
```

Kill 3 pods sequentially with 30s between each:

```bash
viy unveil --eye disintegration --target deployment/nginx \
  --config "podKillCount=3,strategy=sequential,interval=30s"
```

Kill 50% of pods with a 10s grace period:

```bash
viy unveil --eye disintegration --target deployment/nginx \
  --config "podKillPercentage=50,gracePeriod=10s"
```

Preview without executing:

```bash
viy dream --eye disintegration --target deployment/nginx \
  --config "podKillCount=5"
```

Tighten safety constraints:

```bash
viy unveil --eye disintegration --target deployment/nginx \
  --config "podKillCount=3" \
  --blast-radius 20% \
  --min-healthy 4
```

## Execution Flow

1. Resolve the target resource via the Kubernetes API (Deployment, StatefulSet, Service, or Pod)
2. Extract the resource's pod selector, merge with any user-supplied `--selector`
3. List matching pods
4. Calculate kill count (exact or percentage-based)
5. Select pods using the chosen strategy
6. For each selected pod:
   - Log the pod name and namespace
   - Delete the pod with the configured grace period
   - Wait for `interval` before the next deletion (if configured)
   - Respect context cancellation between kills
7. Record a truth: "Revealed N pods in namespace/target"

## Metrics

During and after execution, `Observe()` returns:

| Metric | Description |
|---|---|
| `EyeName` | Always `"disintegration"` |
| `TargetsAffected` | Number of pods successfully deleted |
| `OperationsTotal` | Total delete API calls made |
| `ErrorsTotal` | Delete operations that failed |
| `TruthsRevealed` | Summary of what was revealed |
| `IsActive` | Whether the eye is currently executing |

## Error Handling

| Error | Cause | What to do |
|---|---|---|
| `target not found` | The specified resource does not exist in the cluster | Check the resource name, namespace, and kind |
| `unsupported resource kind` | The target kind is not Pod, Deployment, StatefulSet, or Service | Use a supported kind |
| `insufficient targets` | Requested kill count exceeds available pods | Lower `podKillCount` or check your target selector |
| `invalid configuration` | Both `podKillCount` and `podKillPercentage` set | Use one or the other |
| `blast radius would be exceeded` | Safety guard prevents killing that many pods | Increase `--blast-radius` or decrease kill count |

## See Also

- [Eyes Overview](overview.md) — interface and registry
- [Safety Guards](../safety/guards.md) — blast radius and min healthy constraints
- [CLI Commands](../cli/commands.md) — `unveil` and `dream` usage
