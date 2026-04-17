package death

import (
	"errors"
	"testing"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestDecodeConfig_AllFields(t *testing.T) {
	raw := map[string]any{
		"cpuStressPercent":    75,
		"memoryStressPercent": 50,
		"diskIOBytes":         int64(1024 * 1024),
		"duration":            "5m",
		"rampUp":              "30s",
		"workers":             4,
	}

	result, err := DecodeConfig(raw)
	if err != nil {
		t.Fatalf("DecodeConfig err = %v", err)
	}

	cfg, ok := result.(*Config)
	if !ok {
		t.Fatalf("DecodeConfig returned %T", result)
	}

	if cfg.CPUStressPercent != 75 {
		t.Errorf("CPUStressPercent = %d, want 75", cfg.CPUStressPercent)
	}

	if cfg.MemoryStressPercent != 50 {
		t.Errorf("MemoryStressPercent = %d, want 50", cfg.MemoryStressPercent)
	}

	if cfg.DiskIOBytes != 1024*1024 {
		t.Errorf("DiskIOBytes = %d, want %d", cfg.DiskIOBytes, 1024*1024)
	}

	if cfg.Duration != 5*time.Minute {
		t.Errorf("Duration = %v, want 5m", cfg.Duration)
	}

	if cfg.RampUp != 30*time.Second {
		t.Errorf("RampUp = %v, want 30s", cfg.RampUp)
	}

	if cfg.Workers != 4 {
		t.Errorf("Workers = %d, want 4", cfg.Workers)
	}
}

func TestDecodeConfig_InvalidDuration(t *testing.T) {
	raw := map[string]any{
		"cpuStressPercent": 50,
		"duration":         "eternity",
	}

	_, err := DecodeConfig(raw)
	if err == nil {
		t.Fatal("DecodeConfig should fail on invalid duration")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("err should wrap ErrInvalidConfiguration, got %v", err)
	}
}
