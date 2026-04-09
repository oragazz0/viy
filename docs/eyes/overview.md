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
    Init(podManager PodManager, logger *zap.Logger)
    Unveil(ctx context.Context, target Target, config EyeConfig) error
    Pause(ctx context.Context) error
    Close(ctx context.Context) error
    Observe() Metrics
    Validate(config EyeConfig) error
}
```

### Method Lifecycle

| Method | When Called | Purpose |
|---|---|---|
| `Init` | Before experiment starts | Inject dependencies (pod manager, logger) |
| `Validate` | Before execution | Verify configuration is valid |
| `Unveil` | Experiment start | Execute the chaos injection |
| `Pause` | Manual pause request | Temporarily halt the eye |
| `Close` | Experiment end | Stop and clean up |
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
    TargetsAffected   int       // Number of resources affected
    OperationsTotal   int64     // Total K8s API calls made
    ErrorsTotal       int64     // Errors encountered
    TruthsRevealed    []string  // Human-readable findings
    LastExecutionTime time.Time // Timestamp of last operation
    IsActive          bool      // Whether the eye is currently active
}
```

## Eye Registry

Eyes self-register at import time using Go's `init()` function. The registry lives in `pkg/eyes/registry.go`:

```go
// Registration (in the eye's package)
func init() {
    eyes.Register("disintegration", func() eyes.Eye {
        return &Eye{}
    })
}

// Lookup
eye, err := eyes.Get("disintegration")

// List all registered eyes
names := eyes.List() // returns sorted []string
```

The registry panics on duplicate names — each eye must have a unique identifier.

## Available Eyes

| Eye | Status | Description |
|---|---|---|
| [Disintegration](disintegration.md) | Available | Pod termination — reveals auto-recovery and orchestration health |

## See Also

- [Eye of Disintegration](disintegration.md) — pod kill configuration and examples
- [Extending Viy](../architecture/extending.md) — how to write a new Eye
