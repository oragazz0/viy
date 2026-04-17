package config

import (
	"fmt"
	"sync"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

// Decoder builds a typed eye config from the raw map parsed out of YAML.
// Each eye package registers its own decoder so that the config struct
// stays owned by the eye.
type Decoder func(raw map[string]any) (eyes.EyeConfig, error)

var (
	decoderMu sync.RWMutex
	decoders  = make(map[string]Decoder)
)

// RegisterDecoder registers a decoder for the given eye name.
// Intended to be called from init() alongside eyes.Register.
// Panics if the name is already registered.
func RegisterDecoder(name string, decoder Decoder) {
	decoderMu.Lock()
	defer decoderMu.Unlock()

	if _, exists := decoders[name]; exists {
		panic(fmt.Sprintf("decoder for eye %q already registered", name))
	}

	decoders[name] = decoder
}

// DecodeConfig produces a typed, validated eye config for the named eye
// by dispatching to its registered decoder. Returns an error when the
// eye has no registered decoder or the raw config is malformed.
func DecodeConfig(name string, raw map[string]any) (eyes.EyeConfig, error) {
	decoderMu.RLock()
	decoder, exists := decoders[name]
	decoderMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: no decoder registered for eye %q",
			viyerrors.ErrInvalidConfiguration, name)
	}

	cfg, err := decoder(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding %s config: %w", name, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating %s config: %w", name, err)
	}

	return cfg, nil
}

// HasDecoder reports whether a decoder is registered for the given eye.
func HasDecoder(name string) bool {
	decoderMu.RLock()
	defer decoderMu.RUnlock()

	_, exists := decoders[name]
	return exists
}
