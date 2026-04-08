package errors

import (
	"errors"
	"fmt"
)

var (
	ErrInsufficientPermissions = errors.New("insufficient RBAC permissions")
	ErrBlastRadiusExceeded     = errors.New("blast radius would be exceeded")
	ErrTargetNotFound          = errors.New("target not found")
	ErrInvalidConfiguration    = errors.New("invalid configuration")
	ErrEyesClosed              = errors.New("eyes are closed — no experiment running")
	ErrInsufficientTargets     = errors.New("insufficient targets for requested kill count")
	ErrExperimentNotFound      = errors.New("experiment not found")
	ErrUnsupportedResourceKind = errors.New("unsupported resource kind")
)

// DetailedError wraps an error with actionable context.
type DetailedError struct {
	Err        error
	Suggestion string
	DocsLink   string
}

func (e *DetailedError) Error() string {
	message := e.Err.Error()

	if e.Suggestion != "" {
		message += fmt.Sprintf("\n\n💡 Suggestion: %s", e.Suggestion)
	}

	if e.DocsLink != "" {
		message += fmt.Sprintf("\n📖 Docs: %s", e.DocsLink)
	}

	return message
}

func (e *DetailedError) Unwrap() error {
	return e.Err
}

// WithSuggestion wraps an error with an actionable suggestion.
func WithSuggestion(err error, suggestion string) error {
	return &DetailedError{
		Err:        err,
		Suggestion: suggestion,
	}
}
