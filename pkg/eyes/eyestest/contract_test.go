package eyestest

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/oragazz0/viy/pkg/eyes"
)

func TestRunContractTests_WithCompliantEye(t *testing.T) {
	factory := func(deps eyes.Dependencies) eyes.Eye {
		return &compliantEye{}
	}

	validConfig := &stubConfig{valid: true}
	invalidConfig := &stubConfig{valid: false}

	RunContractTests(t, factory, validConfig, invalidConfig)
}

type compliantEye struct {
	active atomic.Bool
}

func (e *compliantEye) Name() string        { return "compliant" }
func (e *compliantEye) Description() string { return "A compliant test eye" }

func (e *compliantEye) Validate(config eyes.EyeConfig) error {
	return config.Validate()
}

func (e *compliantEye) Unveil(_ context.Context, _ eyes.Target, _ eyes.EyeConfig) error {
	return nil
}

func (e *compliantEye) Pause(_ context.Context) error {
	e.active.Store(false)
	return nil
}

func (e *compliantEye) Close(_ context.Context) error {
	e.active.Store(false)
	return nil
}

func (e *compliantEye) Observe() eyes.Metrics {
	return eyes.Metrics{
		EyeName:  e.Name(),
		IsActive: e.active.Load(),
	}
}

type stubConfig struct {
	valid bool
}

func (c *stubConfig) Validate() error {
	if !c.valid {
		return errors.New("invalid config")
	}
	return nil
}
