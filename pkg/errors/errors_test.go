package errors

import (
	"errors"
	"testing"
)

func TestDetailedError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DetailedError
		contains []string
	}{
		{
			name: "error only",
			err: &DetailedError{
				Err: ErrTargetNotFound,
			},
			contains: []string{"target not found"},
		},
		{
			name: "error with suggestion",
			err: &DetailedError{
				Err:        ErrInsufficientPermissions,
				Suggestion: "apply RBAC config",
			},
			contains: []string{
				"insufficient RBAC permissions",
				"💡 Suggestion: apply RBAC config",
			},
		},
		{
			name: "error with suggestion and docs",
			err: &DetailedError{
				Err:        ErrInvalidConfiguration,
				Suggestion: "check your YAML",
				DocsLink:   "https://docs.example.com",
			},
			contains: []string{
				"invalid configuration",
				"💡 Suggestion: check your YAML",
				"📖 Docs: https://docs.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.err.Error()
			for _, want := range tt.contains {
				if !containsSubstring(message, want) {
					t.Errorf("Error() = %q, want substring %q", message, want)
				}
			}
		})
	}
}

func TestDetailedError_Unwrap(t *testing.T) {
	detailed := &DetailedError{Err: ErrTargetNotFound}

	if !errors.Is(detailed, ErrTargetNotFound) {
		t.Error("Unwrap should allow errors.Is to match the inner error")
	}
}

func TestWithSuggestion(t *testing.T) {
	err := WithSuggestion(ErrBlastRadiusExceeded, "reduce percentage")

	if !errors.Is(err, ErrBlastRadiusExceeded) {
		t.Error("WithSuggestion result should unwrap to original error")
	}

	if !containsSubstring(err.Error(), "reduce percentage") {
		t.Errorf("Error() should contain suggestion, got: %s", err.Error())
	}
}

func containsSubstring(haystack, needle string) bool {
	return len(haystack) >= len(needle) && searchSubstring(haystack, needle)
}

func searchSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}
