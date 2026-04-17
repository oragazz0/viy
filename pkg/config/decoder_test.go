package config

import (
	"errors"
	"fmt"
	"testing"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

type stubConfig struct {
	value      string
	shouldFail bool
}

func (s *stubConfig) Validate() error {
	if s.shouldFail {
		return fmt.Errorf("%w: stub invalid", viyerrors.ErrInvalidConfiguration)
	}

	return nil
}

func TestDecodeConfig_Dispatch(t *testing.T) {
	name := "stub-dispatch"

	RegisterDecoder(name, func(raw map[string]any) (eyes.EyeConfig, error) {
		value, _ := StringField(raw, "value")
		return &stubConfig{value: value}, nil
	})

	result, err := DecodeConfig(name, map[string]any{"value": "hello"})
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}

	stub, ok := result.(*stubConfig)
	if !ok {
		t.Fatalf("DecodeConfig returned %T, want *stubConfig", result)
	}

	if stub.value != "hello" {
		t.Errorf("value = %q, want %q", stub.value, "hello")
	}
}

func TestDecodeConfig_UnknownEye(t *testing.T) {
	_, err := DecodeConfig("never-registered-eye", map[string]any{})
	if err == nil {
		t.Fatal("DecodeConfig should fail for unknown eye")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("error should wrap ErrInvalidConfiguration, got %v", err)
	}
}

func TestDecodeConfig_ValidationFailure(t *testing.T) {
	name := "stub-invalid"

	RegisterDecoder(name, func(_ map[string]any) (eyes.EyeConfig, error) {
		return &stubConfig{shouldFail: true}, nil
	})

	_, err := DecodeConfig(name, map[string]any{})
	if err == nil {
		t.Fatal("DecodeConfig should surface validation failure")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("error should wrap ErrInvalidConfiguration, got %v", err)
	}
}

func TestDecodeConfig_DecoderError(t *testing.T) {
	name := "stub-decoder-error"

	RegisterDecoder(name, func(_ map[string]any) (eyes.EyeConfig, error) {
		return nil, fmt.Errorf("%w: decoder exploded", viyerrors.ErrInvalidConfiguration)
	})

	_, err := DecodeConfig(name, map[string]any{})
	if err == nil {
		t.Fatal("DecodeConfig should surface decoder error")
	}
}

func TestRegisterDecoder_DuplicatePanics(t *testing.T) {
	name := "stub-dup"

	RegisterDecoder(name, func(_ map[string]any) (eyes.EyeConfig, error) {
		return &stubConfig{}, nil
	})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RegisterDecoder should panic on duplicate name")
		}
	}()

	RegisterDecoder(name, func(_ map[string]any) (eyes.EyeConfig, error) {
		return &stubConfig{}, nil
	})
}

func TestHasDecoder(t *testing.T) {
	name := "stub-has"

	if HasDecoder(name) {
		t.Fatalf("HasDecoder(%q) = true before registration", name)
	}

	RegisterDecoder(name, func(_ map[string]any) (eyes.EyeConfig, error) {
		return &stubConfig{}, nil
	})

	if !HasDecoder(name) {
		t.Fatalf("HasDecoder(%q) = false after registration", name)
	}
}
