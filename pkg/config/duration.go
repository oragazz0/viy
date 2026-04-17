package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration wraps time.Duration so that YAML scalars like "5m" or "200ms"
// decode cleanly. The standard time.Duration JSON unmarshaler only accepts
// numeric nanoseconds, which makes for painful YAML.
type Duration time.Duration

// UnmarshalJSON accepts either a quoted duration string ("5m") or a
// numeric value interpreted as nanoseconds.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*d = 0
		return nil
	}

	if data[0] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("duration string: %w", err)
		}

		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", raw, err)
		}

		*d = Duration(parsed)
		return nil
	}

	var ns int64
	if err := json.Unmarshal(data, &ns); err != nil {
		return fmt.Errorf("duration number: %w", err)
	}

	*d = Duration(ns)
	return nil
}

// MarshalJSON emits the duration as a Go duration string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// ToStd returns the underlying time.Duration.
func (d Duration) ToStd() time.Duration {
	return time.Duration(d)
}
