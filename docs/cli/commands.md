# CLI Commands

> Complete reference for all Viy commands and flags.

## Global Flags

These flags apply to all commands:

| Flag | Default | Description |
|---|---|---|
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--output` | `text` | Output format: `text`, `json` (reserved for future use) |
| `--kubeconfig` | — | Path to kubeconfig file |

---

## `viy unveil`

Open an eye — start a chaos experiment.

```bash
viy unveil --eye <name> --target <kind/name> [flags]
```

### Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `--eye` | Yes | — | Eye to open (e.g., `disintegration`) |
| `--target` | Yes | — | Target resource (e.g., `deployment/nginx`) |
| `--namespace` | No | `default` | Kubernetes namespace |
| `--duration` | No | `5m` | How long the experiment runs |
| `--blast-radius` | No | `30%` | Maximum percentage of targets to affect |
| `--config` | No | — | Eye-specific config as `key=value,key=value` |
| `--selector` | No | — | Label selector to filter pods (e.g., `version=v2`) |
| `--dream` | No | `false` | Dry-run mode (same as `viy dream`) |
| `--min-healthy` | No | `1` | Minimum healthy replicas to preserve |

### Target Format

The `--target` flag accepts `kind/name` format. Viy queries the Kubernetes API to fetch the actual resource and extract its pod selector. Supported kinds: `Pod`, `Deployment`, `StatefulSet`, `Service`.

```bash
--target deployment/nginx      # resolves Deployment's .spec.selector → pods
--target statefulset/database  # resolves StatefulSet's .spec.selector → pods
--target service/api           # resolves Service's .spec.selector → pods
--target pod/api-abc           # resolves the Pod's labels → matching pods
```

If the resource does not exist, Viy fails with a `target not found` error before any chaos is executed.

### Combining `--target` and `--selector`

The `--selector` flag adds extra label filtering on top of the resource's own selector. Both are merged:

```bash
# Only affect v2 pods within the api-server Deployment
viy unveil --eye disintegration --target deployment/api-server --selector "version=v2"
```

### Examples

```bash
# Basic experiment — pod termination
viy unveil --eye disintegration --target deployment/nginx

# Custom namespace and duration
viy unveil --eye disintegration --target deployment/api \
  --namespace staging --duration 10m

# With eye-specific config
viy unveil --eye disintegration --target deployment/nginx \
  --config "podKillCount=3,interval=15s"

# Filter by label selector
viy unveil --eye disintegration --target deployment/api \
  --selector "version=v2,tier=backend"

# Tighter safety constraints
viy unveil --eye disintegration --target deployment/nginx \
  --blast-radius 10% --min-healthy 5

# Resource exhaustion — CPU and memory stress
viy unveil --eye death --target deployment/api-server \
  --config "cpuStress=80,memoryStress=70,workers=4,duration=2m"

# Resource exhaustion with ramp-up and disk I/O
viy unveil --eye death --target deployment/worker \
  --config "cpuStress=80,memoryStress=70,diskIOBytes=1048576,duration=2m,rampUp=30s,workers=4"
```

### Signal Handling

`unveil` listens for `SIGINT` (Ctrl+C) and `SIGTERM`. When received, the current operation completes and the experiment stops gracefully.

---

## `viy awaken`

Open **many** eyes at once — run a multi-eye experiment from a YAML file.

```bash
viy awaken --file experiment.yaml [--dream]
```

### Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `--file` | Yes | — | Path to the experiment YAML |
| `--dream` | No | `false` | Dry-run mode — preview without executing |

### Behavior

Viy loads the YAML, validates its structure, decodes each eye's config, resolves every target, checks blast radius per eye, then launches all eyes concurrently. Shared cancellation: `SIGINT` / `SIGTERM` and the wall-clock `spec.duration` propagate to every eye at once.

Each eye's `Close` always runs on a **fresh 30s context** — graceful cleanup survives both parent cancellation and sibling failures (critical for Charm/Death, which run `ExecInContainer` during `Close`).

### Failure Policies

Set via `spec.failurePolicy`:

| Policy | Behavior |
|---|---|
| `continue` *(default)* | Sibling eyes keep running when one errors. All errors are aggregated via `errors.Join` and returned when the experiment ends. |
| `fail-fast` | First eye error cancels every sibling through a shared context. `Close` still runs for every launched eye. |

Both policies share the same wall-clock deadline and signal propagation.

### Contention Handling

If two eyes target the same pod, Viy logs a warning by default. Set `spec.strictIsolation: true` to reject overlap before any eye opens. Detection is based on pod UID — pods recreated mid-experiment don't generate false-positive overlaps from name reuse.

> **Known limitation:** Contention is evaluated once at launch. Pods spawned after the experiment starts are not re-checked.

### Examples

```bash
# Dry-run — resolves targets, checks blast radius, prints plan per eye
viy awaken --file experiment.yaml --dream

# Live multi-eye experiment
viy awaken --file examples/awaken/disintegration-and-charm.yaml
```

See [Experiment YAML](../configuration/experiment-yaml.md) for the full schema and [examples/awaken/disintegration-and-charm.yaml](../../examples/awaken/disintegration-and-charm.yaml) for a working sample.

### Signal Handling

`awaken` listens for `SIGINT` (Ctrl+C) and `SIGTERM`. Cancellation propagates to every in-flight eye at once, and `Close` runs for each one with its own fresh context.

---

## `viy dream`

Dry-run mode — preview what would happen without executing chaos.

```bash
viy dream --eye <name> --target <kind/name> [flags]
```

### Flags

Same as `unveil` except `--duration` and `--dream` are not available (dream mode has no duration and is always dry-run).

| Flag | Required | Default | Description |
|---|---|---|---|
| `--eye` | Yes | — | Eye to open |
| `--target` | Yes | — | Target resource |
| `--namespace` | No | `default` | Kubernetes namespace |
| `--selector` | No | — | Label selector to filter pods (e.g., `version=v2`) |
| `--blast-radius` | No | `30%` | Maximum percentage of targets to affect |
| `--config` | No | — | Eye-specific config |
| `--min-healthy` | No | `1` | Minimum healthy replicas to preserve |

### Output

```
🔮 Dream Mode: Viy dreams of revelation...

Target resolution:
  Resource: deployment/nginx (default) — found ✓
  Selector: app=nginx
  Pods matched: 10

Targets that would be unveiled:
  • Pod: nginx-abc123 (default)
  • Pod: nginx-def456 (default)
  • Pod: nginx-ghi789 (default)

Estimated blast radius: 30% (3/10 pods)
Safety checks: ✅ All passed
```

---

## `viy vision`

List experiments tracked in the local state file.

```bash
viy vision [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--all` | `false` | Include completed and failed experiments |

### Output

By default, only active (`unveiling`) experiments are shown:

```
ID            EYES               STATUS      TARGET          STARTED
a1b2c3d4e5f6  [disintegration]   unveiling   default/nginx   2m30s ago
```

With `--all`:

```
ID            EYES               STATUS      TARGET          STARTED
a1b2c3d4e5f6  [disintegration]   revealed    default/nginx   15m0s ago
b2c3d4e5f6a1  [disintegration]   failed      default/api     1h2m0s ago
```

When no experiments exist: `No experiments found. Viy sleeps.`

---

## `viy slumber`

Stop active experiments.

```bash
viy slumber [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--all` | `false` | Stop all active experiments |
| `--experiment` | — | Stop a specific experiment by ID |
| `--force` | `false` | Force stop (reserved for future use) |

### Behavior

Slumber updates the experiment status to `revealed` in the local state file (`~/.viy/state.json`).

> **Known limitation:** Slumber only updates the state file. It does not cancel experiments running in other processes. If a `viy unveil` process is actively running, it will continue until its duration expires or it receives a signal.

### Examples

```bash
# Stop all active experiments
viy slumber --all

# Stop a specific experiment
viy slumber --experiment a1b2c3d4e5f6
```

---

## `viy version`

Show build information.

```bash
viy version
```

### Output

```
👁️  Viy
  Version: 0.1.0
  Commit:  a1b2c3d
  Built:   2026-04-07T12:00:00Z
```

Values default to `dev` / `none` / `unknown` when not injected via ldflags at build time.

## See Also

- [CLI Flags](../configuration/cli-flags.md) — detailed flag reference and config format
- [Eye of Disintegration](../eyes/disintegration.md) — config options for the disintegration eye
- [Eye of Death](../eyes/death.md) — config options for the death eye (resource exhaustion)
- [Safety Guards](../safety/guards.md) — how blast radius and min healthy work
