package safety

import (
	"fmt"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

// BlastRadiusConfig holds safety limits.
type BlastRadiusConfig struct {
	MaxPercentage      int
	MinHealthyReplicas int
}

// CalculateMaxAffected returns the maximum number of targets that may be
// affected without exceeding the blast radius or violating the minimum
// healthy replicas constraint.
func CalculateMaxAffected(totalTargets int, config BlastRadiusConfig) (int, error) {
	maxAffected := totalTargets * config.MaxPercentage / 100

	if maxAffected == 0 && totalTargets > 0 {
		maxAffected = 1
	}

	remaining := totalTargets - maxAffected
	if remaining < config.MinHealthyReplicas {
		return 0, fmt.Errorf(
			"%w: %d targets minus %d affected leaves %d, below minimum %d",
			viyerrors.ErrBlastRadiusExceeded,
			totalTargets, maxAffected, remaining, config.MinHealthyReplicas,
		)
	}

	return maxAffected, nil
}
