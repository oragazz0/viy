package config

import (
	"fmt"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

// Field extracts a typed value from a raw YAML map with a clear error
// when the shape is wrong. Used by per-eye decoders.
//
// These helpers tolerate missing keys (return the zero value, no error),
// so optional fields stay optional. When the key exists but the value has
// the wrong shape, they return an ErrInvalidConfiguration error.

// StringField returns the string value at key, or "" if absent.
func StringField(raw map[string]any, key string) (string, error) {
	value, exists := raw[key]
	if !exists {
		return "", nil
	}

	typed, ok := value.(string)
	if !ok {
		return "", fieldTypeError(key, "string", value)
	}

	return typed, nil
}

// IntField returns the int value at key, or 0 if absent.
// YAML numbers parse as float64 or int depending on the library;
// this handles both.
func IntField(raw map[string]any, key string) (int, error) {
	value, exists := raw[key]
	if !exists {
		return 0, nil
	}

	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	default:
		return 0, fieldTypeError(key, "integer", value)
	}
}

// Int64Field returns the int64 value at key, or 0 if absent.
func Int64Field(raw map[string]any, key string) (int64, error) {
	value, exists := raw[key]
	if !exists {
		return 0, nil
	}

	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	case float64:
		return int64(typed), nil
	default:
		return 0, fieldTypeError(key, "integer", value)
	}
}

// FloatField returns the float64 value at key, or 0 if absent.
func FloatField(raw map[string]any, key string) (float64, error) {
	value, exists := raw[key]
	if !exists {
		return 0, nil
	}

	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	default:
		return 0, fieldTypeError(key, "number", value)
	}
}

// DurationField parses a duration string like "5m" or "200ms" at key,
// or returns 0 if absent.
func DurationField(raw map[string]any, key string) (time.Duration, error) {
	value, exists := raw[key]
	if !exists {
		return 0, nil
	}

	asString, ok := value.(string)
	if !ok {
		return 0, fieldTypeError(key, "duration string", value)
	}

	parsed, err := time.ParseDuration(asString)
	if err != nil {
		return 0, fmt.Errorf("%w: field %q invalid duration %q: %w",
			viyerrors.ErrInvalidConfiguration, key, asString, err)
	}

	return parsed, nil
}

func fieldTypeError(key, expected string, value any) error {
	return fmt.Errorf("%w: field %q expected %s, got %T",
		viyerrors.ErrInvalidConfiguration, key, expected, value)
}
