package eyes

import (
	"context"
	"time"
)

// Eye represents a chaos module that reveals truths about infrastructure.
type Eye interface {
	Name() string
	Description() string
	Unveil(ctx context.Context, target Target, config EyeConfig) error
	Pause(ctx context.Context) error
	Close(ctx context.Context) error
	Observe() Metrics
	Validate(config EyeConfig) error
}

// Target represents the Kubernetes resource to unveil.
type Target struct {
	Kind      string
	Name      string
	Namespace string
	Labels    map[string]string
	Selector  string
}

// EyeConfig carries eye-specific configuration.
type EyeConfig interface {
	Validate() error
}

// Metrics captures what an eye observes during an experiment.
type Metrics struct {
	TargetsAffected   int
	OperationsTotal   int64
	ErrorsTotal       int64
	TruthsRevealed    []string
	LastExecutionTime time.Time
	IsActive          bool
}
