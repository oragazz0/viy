package disintegration

import (
	"fmt"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

// Config holds pod-kill parameters.
type Config struct {
	PodKillCount      int           `yaml:"podKillCount"`
	PodKillPercentage int           `yaml:"podKillPercentage"`
	Interval          time.Duration `yaml:"interval"`
	Strategy          string        `yaml:"strategy"`
	GracePeriod       time.Duration `yaml:"gracePeriod"`
}

func (c *Config) Validate() error {
	if c.PodKillCount <= 0 && c.PodKillPercentage <= 0 {
		return fmt.Errorf("%w: must specify podKillCount or podKillPercentage",
			viyerrors.ErrInvalidConfiguration)
	}

	if c.PodKillCount > 0 && c.PodKillPercentage > 0 {
		return fmt.Errorf("%w: cannot specify both podKillCount and podKillPercentage",
			viyerrors.ErrInvalidConfiguration)
	}

	if c.PodKillPercentage > 100 {
		return fmt.Errorf("%w: podKillPercentage must be between 1 and 100",
			viyerrors.ErrInvalidConfiguration)
	}

	validStrategies := map[string]bool{"random": true, "sequential": true, "": true}
	if !validStrategies[c.Strategy] {
		return fmt.Errorf("%w: strategy must be 'random' or 'sequential'",
			viyerrors.ErrInvalidConfiguration)
	}

	return nil
}
