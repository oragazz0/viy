package disintegration

import (
	"github.com/oragazz0/viy/pkg/config"
	"github.com/oragazz0/viy/pkg/eyes"
)

func init() {
	config.RegisterDecoder("disintegration", DecodeConfig)
}

// DecodeConfig builds a typed Config from a raw YAML map.
// Validation is performed by the caller via eyes.EyeConfig.Validate.
func DecodeConfig(raw map[string]any) (eyes.EyeConfig, error) {
	cfg := &Config{}

	podKillCount, err := config.IntField(raw, "podKillCount")
	if err != nil {
		return nil, err
	}

	podKillPercentage, err := config.IntField(raw, "podKillPercentage")
	if err != nil {
		return nil, err
	}

	interval, err := config.DurationField(raw, "interval")
	if err != nil {
		return nil, err
	}

	strategy, err := config.StringField(raw, "strategy")
	if err != nil {
		return nil, err
	}

	gracePeriod, err := config.DurationField(raw, "gracePeriod")
	if err != nil {
		return nil, err
	}

	cfg.PodKillCount = podKillCount
	cfg.PodKillPercentage = podKillPercentage
	cfg.Interval = interval
	cfg.Strategy = strategy
	cfg.GracePeriod = gracePeriod

	return cfg, nil
}
