# Architecture

> How Viy's components fit together.

## High-Level Flow

```
User
 │
 ▼
CLI (Cobra)
 │  Parses flags, validates namespace, builds RunConfig
 │
 ▼
Orchestrator
 │  Resolves eye, validates config, resolves targets,
 │  calculates blast radius, manages experiment lifecycle
 │
 ├──► Target Resolver (internal/k8s)
 │    Fetches the K8s resource (Deployment, StatefulSet, Service, Pod),
 │    extracts its pod selector, merges with --selector, returns pods
 │
 ├──► Safety (pkg/safety)
 │    Blast radius calculation, min healthy check
 │
 ├──► Eye Registry (pkg/eyes)
 │    Looks up eye by name, creates instance
 │
 ├──► Eye (internal/eyes/*)
 │    Executes chaos (pod deletion, etc.)
 │    │
 │    └──► K8s Client (internal/k8s)
 │         GetPods, DeletePod via client-go
 │
 └──► State Store (internal/state)
      Persists experiment status to ~/.viy/state.json
```

## Package Layout

Viy follows Go's `pkg` vs `internal` convention:

### `pkg/` — Public API

Packages under `pkg/` define contracts and shared types. They are safe to import from external code.

| Package | Purpose |
|---|---|
| `pkg/eyes` | `Eye` interface, `Target`, `Metrics`, `EyeConfig`, registry |
| `pkg/safety` | `CalculateMaxAffected` — blast radius logic |
| `pkg/errors` | Sentinel errors and `DetailedError` with suggestions |

### `internal/` — Implementation

Packages under `internal/` contain the actual implementations. They cannot be imported by external code.

| Package | Purpose |
|---|---|
| `internal/cli` | Cobra commands: `unveil`, `dream`, `slumber`, `vision`, `version` |
| `internal/orchestrator` | Experiment lifecycle: resolve, validate, execute, persist |
| `internal/eyes/disintegration` | Pod termination eye implementation |
| `internal/k8s` | Kubernetes client-go wrapper (`PodManager`, `TargetResolver` implementations) |
| `internal/state` | JSON file-based experiment persistence |
| `internal/observability` | zap logger factory |
| `internal/version` | Build-time version variables |

### `cmd/` — Entry Point

`cmd/viy/main.go` calls `cli.Execute()`. Nothing else.

## Dependency Direction

Dependencies flow inward — concrete implementations depend on abstractions, never the reverse:

```
cmd/viy → internal/cli → internal/orchestrator → pkg/eyes (interface)
                                                → pkg/safety
                                                → internal/k8s (TargetResolver, PodManager)
                                                → internal/state

internal/k8s (Resolver)      → pkg/eyes (Target type)
                             → pkg/errors

internal/eyes/disintegration → pkg/eyes (implements interface)
                             → pkg/errors
```

The orchestrator depends on `pkg/eyes.Eye` (the interface) and `internal/k8s.TargetResolver` (the interface), not on any specific implementation. Eyes receive their dependencies (e.g., `PodManager`, logger) at construction time via `eyes.Dependencies` — the orchestrator passes this to `eyes.Get()` and the factory injects what each eye needs. Eyes are not aware of target resolution; the `TargetResolver` handles fetching the K8s resource, extracting its selector, and resolving pods.

## Key Design Decisions

### Why client-go Instead of controller-runtime?

Viy is a CLI tool, not a Kubernetes operator. It doesn't need reconciliation loops, watches, or CRD management. `client-go` provides direct API access with less overhead.

### Why Registry Pattern Instead of Plugins?

Eyes self-register via `init()` functions. This is simpler than plugin-based architectures and sufficient for a monorepo. Plugin support may come in v2.0+.

### Why Local State Instead of ConfigMaps?

Keeping state local (`~/.viy/state.json`) means Viy works without cluster-side permissions beyond pod CRUD. CRD-based state is planned for v2.0+.

### Why No YAML Config Loading?

v0.1.0 prioritizes the CLI-first workflow. YAML configuration is planned for v0.3.0 to support reproducible experiment definitions.

## See Also

- [State Persistence](state.md) — how experiments are tracked
- [Extending Viy](extending.md) — adding new eyes
- [Eyes Overview](../eyes/overview.md) — the Eye interface contract
