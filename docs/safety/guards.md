# Safety Guards

> How Viy prevents chaos experiments from causing unintended damage.

## Overview

Viy enforces safety at multiple layers before any pod is touched. These guards are **not configurable** — they are hard-coded constraints that cannot be bypassed.

## Protected Namespaces

The following namespaces are blocked from all experiments:

- `kube-system`
- `kube-public`
- `kube-node-lease`

Attempting to target a protected namespace fails immediately:

```
Error: namespace "kube-system" is protected — chaos experiments are not allowed in system namespaces
```

This check happens at the CLI layer, before the orchestrator or any eye is invoked.

## Blast Radius

The blast radius limits **how many pods** an experiment can affect, expressed as a percentage of total matched pods.

### Calculation

Given:
- `totalTargets` — number of pods matching the label selector
- `maxPercentage` — the `--blast-radius` value (default: 30%)
- `minHealthyReplicas` — the `--min-healthy` value (default: 1)

```
maxAffected = totalTargets * maxPercentage / 100

// Ensure at least 1 pod can be affected
if maxAffected == 0 and totalTargets > 0:
    maxAffected = 1

// Check minimum healthy constraint
remaining = totalTargets - maxAffected
if remaining < minHealthyReplicas:
    ERROR: blast radius would be exceeded
```

### Examples

| Total Pods | Blast Radius | Min Healthy | Max Affected | Result |
|---|---|---|---|---|
| 10 | 30% | 1 | 3 | Allowed (7 remaining >= 1) |
| 10 | 50% | 1 | 5 | Allowed (5 remaining >= 1) |
| 10 | 30% | 8 | 3 | **Blocked** (7 remaining < 8) |
| 3 | 30% | 1 | 1 | Allowed (2 remaining >= 1) |
| 3 | 30% | 3 | 1 | **Blocked** (2 remaining < 3) |
| 1 | 30% | 1 | 1 | **Blocked** (0 remaining < 1) |

### Error Output

When blast radius is exceeded:

```
Error: blast radius would be exceeded: 10 targets minus 3 affected leaves 7, below minimum 8
```

## Minimum Healthy Replicas

The `--min-healthy` flag (default: `1`) sets the floor for how many pods must remain untouched after the experiment. This works in conjunction with blast radius — even if the blast radius allows killing 5 pods, the min healthy constraint may reduce that number.

## Dream Mode as Safety Preview

The `dream` command (or `--dream` flag) runs the full safety pipeline — namespace check, blast radius calculation, target resolution — without executing any chaos. Use it to verify safety before running live experiments.

```bash
viy dream --eye disintegration --target deployment/nginx --blast-radius 50%
```

## Signal Handling

The `unveil` command registers handlers for `SIGINT` and `SIGTERM`. When interrupted:

1. The current inter-kill interval is cancelled
2. No further pods are deleted
3. The experiment is marked as completed in the state file
4. The process exits cleanly

## Safety Enforcement Points

| Layer | Guard | Location |
|---|---|---|
| CLI | Protected namespace check | `internal/cli/unveil.go` |
| CLI | Blast radius percentage parsing (1-100) | `internal/cli/unveil.go` |
| Orchestrator | Blast radius calculation | `pkg/safety/blast_radius.go` |
| Orchestrator | Min healthy replicas check | `pkg/safety/blast_radius.go` |
| Orchestrator | Eye config validation | `internal/orchestrator/orchestrator.go` |
| Eye | Context cancellation between kills | `internal/eyes/disintegration/eye.go` |

## See Also

- [Quickstart](../getting-started/quickstart.md) — see safety in action
- [CLI Commands](../cli/commands.md) — blast radius and min healthy flags
- [Troubleshooting](../runbooks/troubleshooting.md) — resolving safety errors
