package charm

import (
	"errors"
	"testing"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestDecodeConfig_AllFields(t *testing.T) {
	raw := map[string]any{
		"latency":    "200ms",
		"jitter":     "50ms",
		"packetLoss": 0.5,
		"corruption": 1.25,
		"duration":   "2m",
		"interface":  "eth0",
	}

	result, err := DecodeConfig(raw)
	if err != nil {
		t.Fatalf("DecodeConfig err = %v", err)
	}

	cfg, ok := result.(*Config)
	if !ok {
		t.Fatalf("DecodeConfig returned %T", result)
	}

	if cfg.Latency != 200*time.Millisecond {
		t.Errorf("Latency = %v, want 200ms", cfg.Latency)
	}

	if cfg.Jitter != 50*time.Millisecond {
		t.Errorf("Jitter = %v, want 50ms", cfg.Jitter)
	}

	if cfg.PacketLoss != 0.5 {
		t.Errorf("PacketLoss = %v, want 0.5", cfg.PacketLoss)
	}

	if cfg.Corruption != 1.25 {
		t.Errorf("Corruption = %v, want 1.25", cfg.Corruption)
	}

	if cfg.Duration != 2*time.Minute {
		t.Errorf("Duration = %v, want 2m", cfg.Duration)
	}

	if cfg.Interface != "eth0" {
		t.Errorf("Interface = %q, want eth0", cfg.Interface)
	}
}

func TestDecodeConfig_InvalidDuration(t *testing.T) {
	raw := map[string]any{
		"latency": "not-a-duration",
	}

	_, err := DecodeConfig(raw)
	if err == nil {
		t.Fatal("DecodeConfig should fail on invalid duration")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("err should wrap ErrInvalidConfiguration, got %v", err)
	}
}
