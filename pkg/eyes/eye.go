// Package eyes defines the core contract for chaos modules in Viy.
//
// Every chaos module ("eye") implements the [Eye] interface and registers
// itself via [Register] during init(). The orchestrator discovers eyes
// through the global registry and manages their lifecycle.
//
// Eyes receive infrastructure dependencies at construction time through
// [Dependencies], following the dependency injection pattern. Each eye
// uses only the subset of dependencies it needs (interface segregation).
package eyes

import (
	"context"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// PodManager abstracts pod operations for eyes that target pods.
// Satisfied by the internal k8s.Client implementation.
type PodManager interface {
	GetPods(ctx context.Context, namespace, selector string) ([]corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error
}

// Dependencies carries infrastructure dependencies available to eyes.
// Each eye uses only the subset it needs. Extend this struct when new
// eye types require additional infrastructure capabilities.
type Dependencies struct {
	PodManager PodManager
	Logger     *zap.Logger
}

// Eye represents a chaos module that reveals truths about infrastructure.
// Every eye implementation must satisfy this contract to participate in
// single-eye and multi-eye experiments.
//
// Eyes are created via [EyeFactory] functions registered in the global
// registry. Dependencies are injected at construction time — there is
// no separate initialization step.
//
// Lifecycle: factory creates → Validate → Unveil → (Pause/resume) → Close.
type Eye interface {
	// Name returns the unique identifier for this eye type
	// (e.g. "disintegration", "charm"). Must be stable across versions.
	Name() string

	// Description returns a human-readable summary of what this eye reveals.
	Description() string

	// Validate checks whether the given config is valid for this eye.
	// Called before any chaos operations begin. Must not modify state.
	Validate(config EyeConfig) error

	// Unveil executes the chaos operation against the target using the
	// provided config. Blocks until the operation completes or ctx is
	// cancelled. Implementations must respect context cancellation for
	// graceful shutdown.
	Unveil(ctx context.Context, target Target, config EyeConfig) error

	// Pause temporarily halts the chaos operation. The eye remains
	// initialized and can be resumed via another Unveil call.
	Pause(ctx context.Context) error

	// Close permanently stops the eye and releases all resources.
	// After Close returns, the eye must not be reused.
	Close(ctx context.Context) error

	// Observe returns a snapshot of the eye's current metrics.
	// Safe to call concurrently with Unveil.
	Observe() Metrics
}

// Target represents the Kubernetes resource to unveil.
type Target struct {
	Kind      string
	Name      string
	Namespace string
	Labels    map[string]string
	Selector  string
}

// EyeConfig carries eye-specific configuration. Each eye defines its own
// concrete config type that implements this interface.
type EyeConfig interface {
	Validate() error
}

// Metrics captures what an eye observes during an experiment.
// Thread-safe reads are the caller's responsibility — eyes return
// a snapshot from atomic values.
type Metrics struct {
	EyeName           string
	TargetsAffected   int
	OperationsTotal   int64
	ErrorsTotal       int64
	TruthsRevealed    []string
	LastExecutionTime time.Time
	IsActive          bool
}
