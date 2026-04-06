package observability

import (
	"testing"
)

func TestNewLogger_ValidLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			logger, err := NewLogger(level)
			if err != nil {
				t.Fatalf("NewLogger(%q) error = %v", level, err)
			}

			if logger == nil {
				t.Fatal("NewLogger() returned nil logger")
			}
		})
	}
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	_, err := NewLogger("invalid")
	if err == nil {
		t.Fatal("NewLogger() should fail with invalid level")
	}
}
