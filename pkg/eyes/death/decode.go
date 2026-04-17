package death

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

	cpu, err := config.IntField(raw, "cpuStressPercent")
	if err != nil {
		return nil, err
	}

	memory, err := config.IntField(raw, "memoryStressPercent")
	if err != nil {
		return nil, err
	}

	diskIO, err := config.Int64Field(raw, "diskIOBytes")
	if err != nil {
		return nil, err
	}

	duration, err := config.DurationField(raw, "duration")
	if err != nil {
		return nil, err
	}

	rampUp, err := config.DurationField(raw, "rampUp")
	if err != nil {
		return nil, err
	}

	workers, err := config.IntField(raw, "workers")
	if err != nil {
		return nil, err
	}

	cfg.CPUStressPercent = cpu
	cfg.MemoryStressPercent = memory
	cfg.DiskIOBytes = diskIO
	cfg.Duration = duration
	cfg.RampUp = rampUp
	cfg.Workers = workers

	return cfg, nil
}
