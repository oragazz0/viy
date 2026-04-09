package eyes

import (
	"context"
	"testing"
)

func TestRegister_And_Get(t *testing.T) {
	cleanup := isolateRegistry()
	defer cleanup()

	Register("test-eye", func(_ Dependencies) Eye {
		return &stubEye{name: "test-eye"}
	})

	eye, err := Get("test-eye", Dependencies{})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if eye.Name() != "test-eye" {
		t.Errorf("Name() = %q, want %q", eye.Name(), "test-eye")
	}
}

func TestGet_UnknownEye(t *testing.T) {
	cleanup := isolateRegistry()
	defer cleanup()

	_, err := Get("nonexistent", Dependencies{})
	if err == nil {
		t.Fatal("Get should return error for unknown eye")
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	cleanup := isolateRegistry()
	defer cleanup()

	Register("dup", func(_ Dependencies) Eye { return &stubEye{} })

	defer func() {
		if r := recover(); r == nil {
			t.Error("Register should panic on duplicate name")
		}
	}()

	Register("dup", func(_ Dependencies) Eye { return &stubEye{} })
}

func TestList(t *testing.T) {
	cleanup := isolateRegistry()
	defer cleanup()

	Register("charlie", func(_ Dependencies) Eye { return &stubEye{} })
	Register("alpha", func(_ Dependencies) Eye { return &stubEye{} })
	Register("bravo", func(_ Dependencies) Eye { return &stubEye{} })

	names := List()
	if len(names) != 3 {
		t.Fatalf("List() returned %d names, want 3", len(names))
	}

	if names[0] != "alpha" || names[1] != "bravo" || names[2] != "charlie" {
		t.Errorf("List() = %v, want sorted [alpha bravo charlie]", names)
	}
}

func TestExists(t *testing.T) {
	cleanup := isolateRegistry()
	defer cleanup()

	Register("present", func(_ Dependencies) Eye { return &stubEye{} })

	if !Exists("present") {
		t.Error("Exists() should return true for registered eye")
	}

	if Exists("absent") {
		t.Error("Exists() should return false for unregistered eye")
	}
}

// --- helpers ---

func isolateRegistry() func() {
	original := registry

	registryMu.Lock()
	registry = make(map[string]EyeFactory)
	registryMu.Unlock()

	return func() {
		registryMu.Lock()
		registry = original
		registryMu.Unlock()
	}
}

type stubEye struct {
	name string
}

func (s *stubEye) Name() string                                          { return s.name }
func (s *stubEye) Description() string                                   { return "" }
func (s *stubEye) Unveil(_ context.Context, _ Target, _ EyeConfig) error { return nil }
func (s *stubEye) Pause(_ context.Context) error                         { return nil }
func (s *stubEye) Close(_ context.Context) error                         { return nil }
func (s *stubEye) Observe() Metrics                                      { return Metrics{} }
func (s *stubEye) Validate(_ EyeConfig) error                            { return nil }
