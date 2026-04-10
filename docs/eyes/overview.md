# Eyes Overview

> Eyes are Viy's chaos modules. Each eye reveals a different truth about your infrastructure.

## What is an Eye?

An Eye is a self-contained chaos module that implements a specific failure injection strategy. The name comes from Slavic folklore — Viy's servants lift its heavy eyelids so its gaze can reveal hidden truths. In Viy, each Eye "opens" to expose weaknesses in your Kubernetes workloads.

## The Eye Interface

Every Eye implements this contract (defined in `pkg/eyes/eye.go`):

```go
type Eye interface {
    Name() string
    Description() string
    Validate(config EyeConfig) error
    Unveil(ctx context.Context, target Target, config EyeConfig) error
    Pause(ctx context.Context) error
    Close(ctx context.Context) error
    Observe() Metrics
}
```

Dependencies (pod manager, logger, etc.) are injected at construction time through the `Dependencies` struct — there is no separate `Init` step. See [Eye Registry](#eye-registry) below.

### Method Lifecycle

| Method | When Called | Purpose |
|---|---|---|
| `Validate` | Before execution | Verify configuration is valid |
| `Unveil` | Experiment start | Execute the chaos injection |
| `Pause` | Manual pause request | Temporarily halt the eye |
| `Close` | Experiment end | Stop and clean up (eye must not be reused) |
| `Observe` | Any time | Return current metrics snapshot |

## Supporting Types

### Target

Identifies the Kubernetes resource to experiment on:

```go
type Target struct {
    Kind      string            // Resource kind (e.g., "deployment")
    Name      string            // Resource name (e.g., "nginx")
    Namespace string            // Kubernetes namespace
    Labels    map[string]string // Label set
    Selector  string            // Label selector (e.g., "app=nginx")
}
```

Targets are resolved from the `--target` flag. The format `kind/name` causes Viy to query the Kubernetes API for the actual resource (Deployment, StatefulSet, Service, or Pod) and extract its pod selector. An optional `--selector` flag can add extra label filtering on top of the resource's selector.

### EyeConfig

Each Eye defines its own configuration struct that implements:

```go
type EyeConfig interface {
    Validate() error
}
```

### Metrics

In-memory metrics tracked during an experiment:

```go
type Metrics struct {
    EyeName           string    // Identifies which eye produced these metrics
    TargetsAffected   int       // Number of resources affected
    OperationsTotal   int64     // Total K8s API calls made
    ErrorsTotal       int64     // Errors encountered
    TruthsRevealed    []string  // Human-readable findings
    LastExecutionTime time.Time // Timestamp of last operation
    IsActive          bool      // Whether the eye is currently active
}
```

The `EyeName` field allows multi-eye experiments to attribute metrics to the correct eye.

## Eye Registry

Eyes self-register at import time using Go's `init()` function. The registry lives in `pkg/eyes/registry.go`.

### Dependencies

The `Dependencies` struct carries infrastructure dependencies available to eyes. Each eye uses only the subset it needs:

```go
type Dependencies struct {
    PodManager                PodManager
    EphemeralContainerManager EphemeralContainerManager
    Logger                    *zap.Logger
}
```

Each eye uses only the subset it needs — `PodManager` for pod operations, `EphemeralContainerManager` for injecting sidecar processes (used by the Eye of Death and Eye of Charm). Extend this struct when new eye types require additional infrastructure capabilities.

### Registration and Lookup

```go
// Registration (in the eye's package)
func init() {
    eyes.Register("disintegration", func(deps eyes.Dependencies) eyes.Eye {
        return &Eye{
            podManager: deps.PodManager,
            logger:     deps.Logger,
        }
    })
}

// Lookup — creates the eye with dependencies in one step
eye, err := eyes.Get("disintegration", deps)

// Discovery
names := eyes.List()              // returns sorted []string
exists := eyes.Exists("charm")    // check without instantiating
```

The registry panics on duplicate names — each eye must have a unique identifier.

### Contract Tests

The `pkg/eyes/eyestest` package provides reusable contract tests that any eye implementation must pass:

```go
func TestContract(t *testing.T) {
    eyestest.RunContractTests(t, myFactory, validConfig, invalidConfig)
}
```

This validates: non-empty Name/Description, Validate accepts/rejects configs, Observe returns the correct EyeName, inactive by default, and Pause/Close behavior.

## Available Eyes

| Eye | Status | Description |
|---|---|---|
| [Disintegration](disintegration.md) | Available | Pod termination — reveals auto-recovery and orchestration health |
| [Death](death.md) | Available | Resource exhaustion — reveals resource limits, HPA scaling, and OOM killer behavior |
| [Charm](charm.md) | Available | Network chaos — reveals network dependencies, timeouts, and circuit breaker behavior |

## See Also

- [Eye of Disintegration](disintegration.md) — pod kill configuration and examples
- [Eye of Death](death.md) — resource exhaustion configuration and examples
- [Eye of Charm](charm.md) — network chaos configuration and examples
- [Extending Viy](../architecture/extending.md) — how to write a new Eye
