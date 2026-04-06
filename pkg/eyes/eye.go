package eyes

import (
	"context"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// PodManager abstracts pod operations for eyes.
type PodManager interface {
	GetPods(ctx context.Context, namespace, selector string) ([]corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error
}

// Eye represents a chaos module that reveals truths about infrastructure.
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
