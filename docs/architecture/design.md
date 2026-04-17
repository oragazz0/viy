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
| `pkg/eyes/charm` | Eye of Charm — network chaos via `tc netem` in ephemeral containers |
| `pkg/eyes/death` | Eye of Death — resource exhaustion via stress-ng ephemeral containers |
| `pkg/config` | Experiment YAML schema + per-eye decoder registry (parallel to `pkg/eyes` registry) |
| `pkg/safety` | `CalculateMaxAffected` — blast radius logic |
| `pkg/errors` | Sentinel errors and `DetailedError` with suggestions |

### `internal/` — Implementation

Packages under `internal/` contain the actual implementations. They cannot be imported by external code.

| Package | Purpose |
|---|---|
| `internal/cli` | Cobra commands: `unveil`, `awaken`, `dream`, `slumber`, `vision`, `version` |
| `internal/orchestrator` | Experiment lifecycle: `Run` (single-eye), `RunMulti` (multi-eye), contention detection |
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

pkg/eyes/death               → pkg/eyes (implements interface)
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

### Why a Parallel Decoder Registry for YAML?

The eye factory registry (`pkg/eyes/registry.go`) maps eye names to factories that build `Eye` instances. When `viy awaken` loads YAML, it needs the *inverse* — a mapping from eye name to a function that turns raw `map[string]any` into the eye's typed `EyeConfig`. A parallel registry in `pkg/config/decoder.go` keeps the typed config owned by the eye package (each eye calls `config.RegisterDecoder` in its own `init()`), instead of forcing `pkg/config` to import every eye.

## Multi-Eye Execution (`viy awaken`)

`Orchestrator.RunMulti` (in `internal/orchestrator/multi.go`) is the concurrency cornerstone for v0.2.0 and the foundation for Apocalypse mode (v0.4).

### Flow

```
awaken --file X.yaml
  signal.NotifyContext(SIGINT, SIGTERM)          → rootCtx
  config.Load + Experiment.Validate              → schema checks
  for each eyeSpec: config.DecodeConfig          → typed EyeConfig + Validate
  orchestrator.NewOrchestrator(...)
  orch.RunMulti(rootCtx, MultiConfig)
    prepareHandles: buildEye, Validate, resolveTarget, checkBlastRadius per eye
    enforceContention: detect pod-UID overlap; warn or reject
    if DryRun: print per-eye plan, return
    persist Experiment{Eyes:[names...], Status:Unveiling}
    runCtx = context.WithTimeout(rootCtx, spec.duration)
    aggregator goroutine: tick every 10s, log per-eye Observe()
    launch (continue OR fail-fast — see below)
    persist final Experiment{Status:Revealed|Failed}
```

### Failure Policies

**`continue`** uses `sync.WaitGroup` with disjoint error slots. Each goroutine writes into its own slot — no shared mutex. After `Wait`, errors are joined via `errors.Join`. Sibling failures do not cancel each other; only the wall-clock deadline and signal cancellation do.

**`fail-fast`** uses `errgroup.WithContext`. The first non-nil return cancels the group context; every sibling Unveil observes cancellation and unwinds. `g.Wait()` returns the first error.

Both policies share the same `runCtx = context.WithTimeout(rootCtx, spec.duration)` so the wall-clock cap and SIGINT always propagate.

### `runOne` Lifecycle Guarantee

Every launched eye gets `Close` called, even on panic, sibling cancellation, or wall-clock expiry. The deferred block uses a **fresh** `context.Background()` with a 30s timeout — never the group context — so cleanup survives cancellation. This is critical for Charm and Death, whose `Close` calls `ExecInContainer` to remove state inside the target pod.

```go
defer func() {
    if r := recover(); r != nil {
        err = fmt.Errorf("eye %s panicked: %v", handle.name, r)
    }
    closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    _ = handle.eye.Close(closeCtx)
}()
```

### Contention Detection

Built once during `enforceContention`, keyed on **`pod.UID`** (not `pod.Name`). Pod recreation mid-experiment produces a new UID, so a freshly-spawned replica landing in overlap territory won't generate a false positive on subsequent evaluations — and contention detection itself is a launch-time snapshot only.

### Shared Helpers

`Run` (single-eye) and `RunMulti` share `buildEye`, `resolveTarget`, and `checkBlastRadius` to avoid divergence.

## See Also

- [State Persistence](state.md) — how experiments are tracked
- [Extending Viy](extending.md) — adding new eyes
- [Eyes Overview](../eyes/overview.md) — the Eye interface contract
- [Experiment YAML](../configuration/experiment-yaml.md) — multi-eye input schema
