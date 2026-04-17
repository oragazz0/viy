# Experiment YAML

> Schema reference for `viy awaken --file experiment.yaml`.

## Overview

A Viy experiment YAML describes one or more eyes opening simultaneously against a Kubernetes cluster. It is the input format for [`viy awaken`](../cli/commands.md#viy-awaken).

See [examples/awaken/disintegration-and-charm.yaml](../../examples/awaken/disintegration-and-charm.yaml) for a working sample.

## Full Example

```yaml
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment
metadata:
  name: disintegration-and-charm
spec:
  duration: 2m                       # wall-clock cap on RunMulti
  failurePolicy: continue            # continue | fail-fast
  staggerInterval: 5s                # 0 = pure parallel
  strictIsolation: false             # true => reject on pod overlap
  safety:
    maxBlastRadius: 30               # percent, per-eye
    minHealthyReplicas: 1
  eyes:
    - name: disintegration
      duration: 1m                   # optional; must be <= spec.duration
      target:
        kind: deployment
        name: api
        namespace: default
      config:
        podKillCount: 1
        interval: 30s
    - name: charm
      target:
        kind: deployment
        name: api
        namespace: default
      config:
        latency: 200ms
        jitter: 50ms
        duration: 2m
        interface: eth0
```

## Top-Level Fields

| Field | Required | Value |
|---|---|---|
| `apiVersion` | Yes | `chaos.viy.io/v1alpha1` — the only accepted version |
| `kind` | Yes | `ChaosExperiment` — the only accepted kind |
| `metadata.name` | Yes | Non-empty string identifying the experiment |
| `spec` | Yes | See below |

## `spec`

| Field | Required | Default | Description |
|---|---|---|---|
| `duration` | Yes | — | Wall-clock cap applied to the entire multi-eye run. Accepts Go duration strings (`5m`, `90s`, `200ms`) or numeric nanoseconds. Must be positive. |
| `failurePolicy` | No | `continue` | `continue` or `fail-fast`. See [Failure Policies](../cli/commands.md#failure-policies). |
| `staggerInterval` | No | `0` | Delay between successive eye launches. `0` means all eyes fire at `t=0`. Must be non-negative. |
| `strictIsolation` | No | `false` | When `true`, Viy rejects the experiment if two eyes' target resolution overlaps on the same pod. |
| `safety` | Yes | — | Per-eye blast radius limits — see below. |
| `eyes` | Yes | — | Non-empty list of eye specs. See [Eye Spec](#eye-spec). |

### `spec.safety`

| Field | Description |
|---|---|
| `maxBlastRadius` | Maximum percentage of target pods each eye may affect (0–100). Applied independently per eye. |
| `minHealthyReplicas` | Minimum healthy replicas Viy will preserve regardless of blast radius. |

See [Safety Guards](../safety/guards.md) for the calculation.

## Eye Spec

Each entry in `spec.eyes` describes one eye's participation:

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | Yes | — | Eye identifier (`disintegration`, `charm`, `death`). Must be unique within the experiment. |
| `duration` | No | `spec.duration` | Optional per-eye cap. Must be `>= 0` and `<= spec.duration`. |
| `target` | Yes | — | Kubernetes resource the eye acts on — see below. |
| `config` | Yes | — | Eye-specific configuration block. See [Per-Eye Config](#per-eye-config). |

### `target`

| Field | Required | Description |
|---|---|---|
| `kind` | Yes | Resource kind: `pod`, `deployment`, `statefulset`, or `service`. |
| `name` | Yes | Resource name. |
| `namespace` | Yes | Kubernetes namespace. Protected namespaces (`kube-system`, `kube-public`, `kube-node-lease`) are rejected before launch. |
| `selector` | No | Extra label selector merged with the resource's own selector. |

## Per-Eye Config

The `config` block for each eye is decoded into a typed struct owned by the eye package. Unknown keys are silently ignored.

### Disintegration

| Key | Type | Description |
|---|---|---|
| `podKillCount` | int | Fixed number of pods to kill per interval |
| `podKillPercentage` | int (1–100) | Percentage of pods to kill instead of a fixed count |
| `interval` | duration | Wait between kills (`30s`, `1m`) |
| `strategy` | string | `random` or `sequential` |
| `gracePeriod` | duration | Grace period passed to pod deletion |

One of `podKillCount` or `podKillPercentage` must be set. See [Eye of Disintegration](../eyes/disintegration.md).

### Charm

| Key | Type | Description |
|---|---|---|
| `latency` | duration | Added request latency (`200ms`) |
| `jitter` | duration | Latency jitter. Requires `latency` to also be set. |
| `packetLoss` | float (0–100) | Packet loss percentage |
| `corruption` | float (0–100) | Packet corruption percentage |
| `duration` | duration | How long the netem rule stays applied (required) |
| `interface` | string | Network interface (typically `eth0`) |

At least one of `latency`, `packetLoss`, or `corruption` must be set. See [Eye of Charm](../eyes/charm.md).

### Death

| Key | Type | Description |
|---|---|---|
| `cpuStressPercent` | int (1–100) | CPU load per worker |
| `memoryStressPercent` | int (1–100) | Memory consumption per worker |
| `diskIOBytes` | int64 | Disk I/O bytes per worker |
| `duration` | duration | How long stress-ng runs (required) |
| `rampUp` | duration | Gradual ramp-up before full stress |
| `workers` | int | Number of stress-ng workers (required) |

At least one of the stress types must be enabled. See [Eye of Death](../eyes/death.md).

## Duration Format

Any field typed as `duration` accepts either:

- A Go duration string: `200ms`, `30s`, `5m`, `1h`, `1h30m`
- Numeric nanoseconds: `1000000000` (equivalent to `1s`)

## Validation Order

Viy validates in this order and aborts on the first error:

1. File exists and parses as YAML
2. Top-level schema (`apiVersion`, `kind`, `metadata.name`, `spec.duration > 0`, `failurePolicy` enum, `spec.eyes` non-empty, no duplicate eye names, per-eye duration within bounds, target fields present)
3. Namespace not in the protected set
4. Each eye's `config` decodes cleanly and its typed config `Validate()` passes
5. Kubernetes targets resolve to concrete pods
6. Per-eye blast radius stays within `safety.maxBlastRadius` and respects `safety.minHealthyReplicas`
7. Contention check (warn by default, reject under `strictIsolation`)

Only after all of the above does any `Unveil` run.

## See Also

- [`viy awaken`](../cli/commands.md#viy-awaken) — CLI invocation
- [Eyes Overview](../eyes/overview.md) — the Eye interface contract
- [Safety Guards](../safety/guards.md) — blast radius and min-healthy calculation
- [Architecture: Multi-Eye Execution](../architecture/design.md#multi-eye-execution-viy-awaken) — concurrency model
