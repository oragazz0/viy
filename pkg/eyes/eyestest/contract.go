// Package eyestest provides reusable contract tests for [eyes.Eye]
// implementations. Any eye that passes these tests is guaranteed to
// satisfy the interface contract expected by the orchestrator and
// multi-eye execution engine.
//
// Usage from an eye's test file:
//
//	func TestContract(t *testing.T) {
//	    eyestest.RunContractTests(t, myFactory, validCfg, invalidCfg)
//	}
package eyestest

import (
	"context"
	"testing"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/oragazz0/viy/pkg/eyes"
)

// RunContractTests exercises the full [eyes.Eye] contract against the
// given factory. It verifies identity, validation, observation, pause,
// and close behaviors. validConfig must pass Validate; invalidConfig
// must fail it.
func RunContractTests(t *testing.T, factory eyes.EyeFactory, validConfig, invalidConfig eyes.EyeConfig) {
	t.Helper()

	deps := makeDeps()

	t.Run("Name_NonEmpty", func(t *testing.T) {
		eye := factory(deps)
		if eye.Name() == "" {
			t.Error("Name() must return a non-empty string")
		}
	})

	t.Run("Description_NonEmpty", func(t *testing.T) {
		eye := factory(deps)
		if eye.Description() == "" {
			t.Error("Description() must return a non-empty string")
		}
	})

	t.Run("Validate_AcceptsValidConfig", func(t *testing.T) {
		eye := factory(deps)
		if err := eye.Validate(validConfig); err != nil {
			t.Errorf("Validate(validConfig) returned error: %v", err)
		}
	})

	t.Run("Validate_RejectsInvalidConfig", func(t *testing.T) {
		eye := factory(deps)
		if err := eye.Validate(invalidConfig); err == nil {
			t.Error("Validate(invalidConfig) should return an error")
		}
	})

	t.Run("Observe_ReturnsEyeName", func(t *testing.T) {
		eye := factory(deps)
		metrics := eye.Observe()

		if metrics.EyeName != eye.Name() {
			t.Errorf("Observe().EyeName = %q, want %q", metrics.EyeName, eye.Name())
		}
	})

	t.Run("Observe_InactiveByDefault", func(t *testing.T) {
		eye := factory(deps)
		metrics := eye.Observe()

		if metrics.IsActive {
			t.Error("newly created eye should not be active")
		}
	})

	t.Run("Pause_NoError", func(t *testing.T) {
		eye := factory(deps)
		if err := eye.Pause(context.Background()); err != nil {
			t.Errorf("Pause() returned error: %v", err)
		}
	})

	t.Run("Close_NoError", func(t *testing.T) {
		eye := factory(deps)
		if err := eye.Close(context.Background()); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	t.Run("Close_SetsInactive", func(t *testing.T) {
		eye := factory(deps)
		_ = eye.Close(context.Background())

		metrics := eye.Observe()
		if metrics.IsActive {
			t.Error("eye should be inactive after Close()")
		}
	})
}

func makeDeps() eyes.Dependencies {
	logger, _ := zap.NewDevelopment()
	return eyes.Dependencies{
		PodManager:                &noopPodManager{},
		EphemeralContainerManager: &noopEphemeralContainerManager{},
		Logger:                    logger,
	}
}

type noopPodManager struct{}

func (n *noopPodManager) GetPods(_ context.Context, _, _ string) ([]corev1.Pod, error) {
	return nil, nil
}

func (n *noopPodManager) DeletePod(_ context.Context, _, _ string, _ int64) error {
	return nil
}

type noopEphemeralContainerManager struct{}

func (n *noopEphemeralContainerManager) AddEphemeralContainer(_ context.Context, _, _ string, _ corev1.EphemeralContainer) error {
	return nil
}

func (n *noopEphemeralContainerManager) ExecInContainer(_ context.Context, _, _, _ string, _ []string) error {
	return nil
}
