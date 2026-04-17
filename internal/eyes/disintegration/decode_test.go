package disintegration

import (
	"errors"
	"testing"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestDecodeConfig_AllFields(t *testing.T) {
	raw := map[string]any{
		"podKillCount": 3,
		"interval":     "30s",
		"strategy":     "sequential",
		"gracePeriod":  "10s",
	}

	result, err := DecodeConfig(raw)
	if err != nil {
		t.Fatalf("DecodeConfig err = %v", err)
	}

	cfg, ok := result.(*Config)
	if !ok {
		t.Fatalf("DecodeConfig returned %T", result)
	}

	if cfg.PodKillCount != 3 {
		t.Errorf("PodKillCount = %d, want 3", cfg.PodKillCount)
	}

	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}

	if cfg.Strategy != "sequential" {
		t.Errorf("Strategy = %q", cfg.Strategy)
	}

	if cfg.GracePeriod != 10*time.Second {
		t.Errorf("GracePeriod = %v", cfg.GracePeriod)
	}
}

func TestDecodeConfig_Percentage(t *testing.T) {
	raw := map[string]any{"podKillPercentage": 25}

	result, err := DecodeConfig(raw)
	if err != nil {
		t.Fatalf("DecodeConfig err = %v", err)
	}

	cfg := result.(*Config)
	if cfg.PodKillPercentage != 25 {
		t.Errorf("PodKillPercentage = %d, want 25", cfg.PodKillPercentage)
	}
}

func TestDecodeConfig_InvalidDuration(t *testing.T) {
	raw := map[string]any{
		"podKillCount": 1,
		"interval":     "not-a-duration",
	}

	_, err := DecodeConfig(raw)
	if err == nil {
		t.Fatal("DecodeConfig should fail on invalid duration")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("err should wrap ErrInvalidConfiguration, got %v", err)
	}
}
