# CLI Flags and Configuration

> How Viy is configured today: CLI flags and the key=value config format.

## Overview

Viy v0.1.0 is configured entirely through CLI flags. There is no YAML config file loading yet.

Configuration flows through two layers:

1. **Global flags** — apply to all commands (log level, kubeconfig, output format)
2. **Command flags** — specific to `unveil` and `dream` (eye, target, blast radius, etc.)
3. **Eye config** — passed as a `--config` string with eye-specific parameters

## Global Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--log-level` | string | `info` | Structured log level. Valid: `debug`, `info`, `warn`, `error` |
| `--output` | string | `text` | Output format. Valid: `text`, `json`. Note: `json` is accepted but not yet implemented |
| `--kubeconfig` | string | — | Explicit path to a kubeconfig file. See [Installation](../getting-started/installation.md#kubeconfig) for resolution order |

## Blast Radius

The `--blast-radius` flag accepts a percentage with or without the `%` suffix:

```bash
--blast-radius 30%   # valid
--blast-radius 30    # also valid
```

Valid range: `1%` to `100%`. Values outside this range are rejected.

The blast radius determines the **maximum percentage of matched pods** that can be affected. See [Safety Guards](../safety/guards.md) for the calculation logic.

## Eye Config Format

The `--config` flag accepts a comma-separated list of `key=value` pairs:

```bash
--config "podKillCount=3,strategy=sequential,interval=30s"
```

### Parsing Rules

- Pairs are split by `,`
- Keys and values are split by `=` (first `=` only — values can contain `=`)
- Leading/trailing whitespace is trimmed from both key and value
- Invalid pairs (missing `=`) are silently ignored
- Unknown keys are silently ignored
- Values are parsed according to their type:
  - **int**: `podKillCount=3`
  - **percentage**: `podKillPercentage=50` (the `%` suffix is optional and stripped)
  - **duration**: `interval=30s`, `gracePeriod=10s` (Go duration format)
  - **string**: `strategy=random`

### Defaults

When `--config` is omitted or empty, the disintegration eye uses:

```
podKillCount=1
strategy=random
interval=0s
gracePeriod=0s
```

### Available Keys (Disintegration)

See [Eye of Disintegration](../eyes/disintegration.md) for the full config reference.

## See Also

- [CLI Commands](../cli/commands.md) — command-specific flags
- [Eye of Disintegration](../eyes/disintegration.md) — config keys for the disintegration eye
