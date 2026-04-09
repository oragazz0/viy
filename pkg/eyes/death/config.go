package death

import (
	"fmt"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

const (
	minStressPercent = 1
	maxStressPercent = 100
	minWorkers = 1
)

// Config holds resource exhaustion parameters for the Eye of Death.
type Config struct {
	CPUStressPercent    int
	MemoryStressPercent int
	DiskIOBytes         int64
	Duration            time.Duration
	RampUp              time.Duration
	Workers             int
}

func (c *Config) Validate() error {
	if c.CPUStressPercent == 0 && c.MemoryStressPercent == 0 && c.DiskIOBytes == 0 {
		return fmt.Errorf(
			"%w: at least one stress type must be enabled (cpuStress, memoryStress, or diskIOBytes)",
			viyerrors.ErrInvalidConfiguration,
		)
	}

	if err := validatePercent("cpuStress", c.CPUStressPercent); err != nil {
		return err
	}

	if err := validatePercent("memoryStress", c.MemoryStressPercent); err != nil {
		return err
	}

	if c.DiskIOBytes < 0 {
		return fmt.Errorf("%w: diskIOBytes must be non-negative", viyerrors.ErrInvalidConfiguration)
	}

	if c.Duration <= 0 {
		return fmt.Errorf("%w: duration must be positive", viyerrors.ErrInvalidConfiguration)
	}

	if c.RampUp < 0 {
		return fmt.Errorf("%w: rampUp must be non-negative", viyerrors.ErrInvalidConfiguration)
	}

	if c.RampUp >= c.Duration {
		return fmt.Errorf("%w: rampUp must be less than duration", viyerrors.ErrInvalidConfiguration)
	}

	if c.Workers < minWorkers {
		return fmt.Errorf("%w: workers must be at least %d", viyerrors.ErrInvalidConfiguration, minWorkers)
	}

	return nil
}

func validatePercent(field string, value int) error {
	if value == 0 {
		return nil
	}

	if value < minStressPercent || value > maxStressPercent {
		return fmt.Errorf(
			"%w: %s must be between %d%% and %d%%",
			viyerrors.ErrInvalidConfiguration, field, minStressPercent, maxStressPercent,
		)
	}

	return nil
}
