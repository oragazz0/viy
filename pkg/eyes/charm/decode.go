package charm

import (
	"github.com/oragazz0/viy/pkg/config"
	"github.com/oragazz0/viy/pkg/eyes"
)

func init() {
	config.RegisterDecoder(eyeName, DecodeConfig)
}

// DecodeConfig builds a typed Config from a raw YAML map.
// Validation is performed by the caller via eyes.EyeConfig.Validate.
func DecodeConfig(raw map[string]any) (eyes.EyeConfig, error) {
	cfg := &Config{}

	latency, err := config.DurationField(raw, "latency")
	if err != nil {
		return nil, err
	}

	jitter, err := config.DurationField(raw, "jitter")
	if err != nil {
		return nil, err
	}

	packetLoss, err := config.FloatField(raw, "packetLoss")
	if err != nil {
		return nil, err
	}

	corruption, err := config.FloatField(raw, "corruption")
	if err != nil {
		return nil, err
	}

	duration, err := config.DurationField(raw, "duration")
	if err != nil {
		return nil, err
	}

	iface, err := config.StringField(raw, "interface")
	if err != nil {
		return nil, err
	}

	cfg.Latency = latency
	cfg.Jitter = jitter
	cfg.PacketLoss = packetLoss
	cfg.Corruption = corruption
	cfg.Duration = duration
	cfg.Interface = iface

	return cfg, nil
}
