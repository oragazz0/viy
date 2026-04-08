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
| `--dream` | No | `false` | Dry-run mode (same as `viy dream`) |
| `--min-healthy` | No | `1` | Minimum healthy replicas to preserve |

### Target Format

The `--target` flag accepts `kind/name` format. The kind is extracted and the name is converted to a label selector `app=<name>`.

```bash
--target deployment/nginx    # selects pods with label app=nginx
--target deployment/api      # selects pods with label app=api
```

### Examples

```bash
# Basic experiment
viy unveil --eye disintegration --target deployment/nginx

# Custom namespace and duration
viy unveil --eye disintegration --target deployment/api \
  --namespace staging --duration 10m

# With eye-specific config
viy unveil --eye disintegration --target deployment/nginx \
  --config "podKillCount=3,interval=15s"

# Tighter safety constraints
viy unveil --eye disintegration --target deployment/nginx \
  --blast-radius 10% --min-healthy 5
```

### Signal Handling

`unveil` listens for `SIGINT` (Ctrl+C) and `SIGTERM`. When received, the current operation completes and the experiment stops gracefully.

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
| `--blast-radius` | No | `30%` | Maximum percentage of targets to affect |
| `--config` | No | — | Eye-specific config |
| `--min-healthy` | No | `1` | Minimum healthy replicas to preserve |

### Output

```
🔮 Dream Mode: Viy dreams of revelation...

Targets that would be unveiled:
  • Pod: nginx-abc123 (default/nginx)
  • Pod: nginx-def456 (default/nginx)

Estimated blast radius: 30% (2/10 pods)
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
- [Safety Guards](../safety/guards.md) — how blast radius and min healthy work
