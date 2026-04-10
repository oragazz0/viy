package charm

import (
	"fmt"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

const (
	minPercent = 0.0
	maxPercent = 100.0
)

// Config holds network chaos parameters for the Eye of Charm.
type Config struct {
	Latency    time.Duration
	Jitter     time.Duration
	PacketLoss float64
	Corruption float64
	Duration   time.Duration
	Interface  string
}

func (c *Config) Validate() error {
	if c.Latency < 0 {
		return fmt.Errorf("%w: latency must be non-negative", viyerrors.ErrInvalidConfiguration)
	}

	if c.Jitter < 0 {
		return fmt.Errorf("%w: jitter must be non-negative", viyerrors.ErrInvalidConfiguration)
	}

	if err := validatePercent("packetLoss", c.PacketLoss); err != nil {
		return err
	}

	if err := validatePercent("corruption", c.Corruption); err != nil {
		return err
	}

	if !hasAnyChaosParameter(c) {
		return fmt.Errorf(
			"%w: at least one chaos parameter must be set (latency, packetLoss, or corruption)",
			viyerrors.ErrInvalidConfiguration,
		)
	}

	if c.Jitter > 0 && c.Latency == 0 {
		return fmt.Errorf("%w: jitter requires latency to be set", viyerrors.ErrInvalidConfiguration)
	}

	if c.Duration <= 0 {
		return fmt.Errorf("%w: duration must be positive", viyerrors.ErrInvalidConfiguration)
	}

	return nil
}

func hasAnyChaosParameter(c *Config) bool {
	return c.Latency > 0 || c.PacketLoss > 0 || c.Corruption > 0
}

func validatePercent(field string, value float64) error {
	if value < minPercent || value > maxPercent {
		return fmt.Errorf(
			"%w: %s must be between %.0f%% and %.0f%%",
			viyerrors.ErrInvalidConfiguration, field, minPercent, maxPercent,
		)
	}

	return nil
}
