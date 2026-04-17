# Extending Viy — Writing a New Eye

> Step-by-step guide to adding a new chaos module.

## Overview

Adding a new Eye requires five steps:

1. Define a config struct implementing `EyeConfig`
2. Implement the `Eye` interface with dependency injection via factory
3. Register via `init()`
4. Register a YAML decoder so the eye works with `viy awaken`
5. Wire up contract tests

The Eye of Death (`pkg/eyes/death/`) serves as the reference implementation.

## Step 1: Create the Package

Create a new directory under `pkg/eyes/`:

```
pkg/eyes/youreyename/
├── eye.go
├── config.go
└── eye_test.go
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

Create `eye.go` implementing the `eyes.Eye` interface. Dependencies are injected at construction time through the factory — there is no `Init` method:

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

// Register the eye at import time. The factory receives Dependencies
// and stores what this eye needs. Use only the subset of dependencies
// your eye requires (interface segregation).
func init() {
    eyes.Register("youreyename", func(deps eyes.Dependencies) eyes.Eye {
        return &Eye{
            podManager: deps.PodManager,
            logger:     deps.Logger,
            // If your eye needs ephemeral container injection:
            // ephemeralContainers: deps.EphemeralContainerManager,
        }
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
        EyeName:           e.Name(),
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

The eye registers itself via `init()`, but the package must be imported somewhere for `init()` to execute. Add a blank import to `internal/cli/eyes_register.go` so both the eye factory and its YAML decoder (Step 5) are loaded when the CLI starts.

## Step 5: Register a YAML Decoder

For your eye to participate in `viy awaken`, register a decoder that builds your typed `Config` from a raw `map[string]any` (the YAML `config` block). Add `decode.go` to your package:

```go
package youreyename

import (
    "github.com/oragazz0/viy/pkg/config"
    "github.com/oragazz0/viy/pkg/eyes"
)

func init() {
    config.RegisterDecoder("youreyename", DecodeConfig)
}

// DecodeConfig builds a typed Config from a raw YAML map.
// Validation is performed by the caller via eyes.EyeConfig.Validate.
func DecodeConfig(raw map[string]any) (eyes.EyeConfig, error) {
    someParameter, err := config.IntField(raw, "someParameter")
    if err != nil {
        return nil, err
    }

    anotherParam, err := config.DurationField(raw, "anotherParam")
    if err != nil {
        return nil, err
    }

    return &Config{
        SomeParameter: someParameter,
        AnotherParam:  anotherParam,
    }, nil
}
```

The `pkg/config` field helpers (`IntField`, `Int64Field`, `FloatField`, `StringField`, `DurationField`) tolerate missing keys (return zero + nil) and wrap type errors in `ErrInvalidConfiguration`. Unknown YAML keys are silently ignored, matching the CLI `--config` parser's behavior.

The eye's `DecodeConfig` returns a **raw typed config**; `config.DecodeConfig` (the dispatcher) then calls `EyeConfig.Validate()` on the result, so you don't need to re-validate inside the decoder.

## Step 6: Wire Contract Tests

Add a contract test to `eye_test.go` to verify your eye satisfies the interface contract. The `pkg/eyes/eyestest` package provides a reusable test suite:

```go
package youreyename

import (
    "testing"

    "github.com/oragazz0/viy/pkg/eyes"
    "github.com/oragazz0/viy/pkg/eyes/eyestest"
)

func TestContract(t *testing.T) {
    factory := func(deps eyes.Dependencies) eyes.Eye {
        return &Eye{
            podManager: deps.PodManager,
            logger:     deps.Logger,
        }
    }

    validConfig := &Config{SomeParameter: 1}
    invalidConfig := &Config{}

    eyestest.RunContractTests(t, factory, validConfig, invalidConfig)
}
```

This validates: non-empty Name/Description, Validate accepts/rejects configs, Observe returns the correct EyeName, inactive by default, and Pause/Close behavior. Any future changes to the `Eye` interface contract will be caught here.

## Step 7: Add CLI Config Parsing

In the CLI layer, add a config builder for your eye's `--config` key=value parsing, following the existing pattern. This is what `viy unveil --eye youreyename --config "..."` uses; `viy awaken` uses the YAML decoder from Step 5.

## Checklist

- [ ] Config struct implements `eyes.EyeConfig` with `Validate()`
- [ ] Eye struct implements all 7 methods of `eyes.Eye`
- [ ] Factory receives `eyes.Dependencies` and stores needed deps
- [ ] `init()` calls `eyes.Register()` with a unique name
- [ ] `init()` also calls `config.RegisterDecoder()` so the eye works with `viy awaken`
- [ ] Blank import added to `internal/cli/eyes_register.go`
- [ ] `Observe()` returns `EyeName` matching `Name()`
- [ ] `Unveil()` respects context cancellation
- [ ] Metrics are updated atomically
- [ ] Errors use sentinel types from `pkg/errors`
- [ ] Contract tests pass via `eyestest.RunContractTests()`
- [ ] Decoder tests cover: all fields, missing fields, invalid duration
- [ ] Tests cover: success path, error paths, config validation, context cancellation
- [ ] CLI config builder parses `--config` key=value pairs

## See Also

- [Eyes Overview](../eyes/overview.md) — interface and registry details
- [Eye of Death](../eyes/death.md) — reference implementation (resource exhaustion)
- [Eye of Charm](../eyes/charm.md) — network chaos (ephemeral container + tc netem)
- [Eye of Disintegration](../eyes/disintegration.md) — pod termination
- [Architecture](design.md) — where eyes fit in the dependency graph
