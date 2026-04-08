# Extending Viy — Writing a New Eye

> Step-by-step guide to adding a new chaos module.

## Overview

Adding a new Eye requires three steps:

1. Implement the `Eye` interface
2. Define a config struct implementing `EyeConfig`
3. Register via `init()`

The existing Eye of Disintegration (`internal/eyes/disintegration/`) serves as the reference implementation.

## Step 1: Create the Package

Create a new directory under `internal/eyes/`:

```
internal/eyes/youreyename/
├── eye.go
├── config.go
├── eye_test.go
└── config_test.go
```

## Step 2: Define the Config

Create `config.go` with a struct that implements `eyes.EyeConfig`:

```go
package youreyename

import (
    "fmt"

    viyerrors "github.com/oragazz0/viy/pkg/errors"
)

type Config struct {
    // Your eye-specific fields here
    SomeParameter int           `yaml:"someParameter"`
    AnotherParam  time.Duration `yaml:"anotherParam"`
}

func (c *Config) Validate() error {
    if c.SomeParameter <= 0 {
        return fmt.Errorf("%w: someParameter must be positive",
            viyerrors.ErrInvalidConfiguration)
    }
    return nil
}
```

## Step 3: Implement the Eye

Create `eye.go` implementing the `eyes.Eye` interface:

```go
package youreyename

import (
    "context"
    "fmt"
    "sync/atomic"
    "time"

    "go.uber.org/zap"

    viyerrors "github.com/oragazz0/viy/pkg/errors"
    "github.com/oragazz0/viy/pkg/eyes"
)

// Register the eye at import time.
func init() {
    eyes.Register("youreyename", func() eyes.Eye {
        return &Eye{}
    })
}

type Eye struct {
    podManager      eyes.PodManager
    logger          *zap.Logger
    targetsAffected atomic.Int64
    operationsTotal atomic.Int64
    errorsTotal     atomic.Int64
    truthsRevealed  []string
    lastExecution   atomic.Int64
    active          atomic.Bool
}

func (e *Eye) Name() string        { return "youreyename" }
func (e *Eye) Description() string { return "Reveals something about your infrastructure" }

func (e *Eye) Init(podManager eyes.PodManager, logger *zap.Logger) {
    e.podManager = podManager
    e.logger = logger
}

func (e *Eye) Validate(config eyes.EyeConfig) error {
    cfg, ok := config.(*Config)
    if !ok {
        return fmt.Errorf("%w: expected *youreyename.Config",
            viyerrors.ErrInvalidConfiguration)
    }
    return cfg.Validate()
}

func (e *Eye) Unveil(ctx context.Context, target eyes.Target, config eyes.EyeConfig) error {
    cfg := config.(*Config) // safe after Validate

    e.active.Store(true)
    defer e.active.Store(false)

    // Your chaos logic here:
    // 1. Resolve targets via e.podManager.GetPods()
    // 2. Apply chaos
    // 3. Update metrics atomically
    // 4. Respect ctx.Done() for cancellation

    return nil
}

func (e *Eye) Pause(_ context.Context) error {
    e.active.Store(false)
    return nil
}

func (e *Eye) Close(_ context.Context) error {
    e.active.Store(false)
    return nil
}

func (e *Eye) Observe() eyes.Metrics {
    return eyes.Metrics{
        TargetsAffected:   int(e.targetsAffected.Load()),
        OperationsTotal:   e.operationsTotal.Load(),
        ErrorsTotal:       e.errorsTotal.Load(),
        TruthsRevealed:    e.truthsRevealed,
        LastExecutionTime: time.Unix(0, e.lastExecution.Load()),
        IsActive:          e.active.Load(),
    }
}
```

## Step 4: Wire the Import

The eye registers itself via `init()`, but the package must be imported somewhere for `init()` to execute. Add a blank import to the CLI layer that builds the config for your eye.

In `internal/cli/unveil.go`, the disintegration eye is imported directly because the CLI builds its config. Follow the same pattern — add your import and a `buildYourEyeConfig()` function.

## Step 5: Add CLI Config Parsing

In `internal/cli/unveil.go`, add a config builder for your eye's `--config` key=value parsing, following the pattern of `buildDisintegrationConfig()`.

## Checklist

- [ ] Config struct implements `eyes.EyeConfig` with `Validate()`
- [ ] Eye struct implements all 8 methods of `eyes.Eye`
- [ ] `init()` calls `eyes.Register()` with a unique name
- [ ] `Unveil()` respects context cancellation
- [ ] Metrics are updated atomically
- [ ] Errors use sentinel types from `pkg/errors`
- [ ] Tests cover: success path, error paths, config validation, context cancellation
- [ ] CLI config builder parses `--config` key=value pairs

## See Also

- [Eyes Overview](../eyes/overview.md) — interface and registry details
- [Eye of Disintegration](../eyes/disintegration.md) — reference implementation
- [Architecture](design.md) — where eyes fit in the dependency graph
