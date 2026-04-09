package eyes

import (
	"fmt"
	"sort"
	"sync"
)

// EyeFactory creates a new Eye instance with the given dependencies.
// Registered via [Register] during init() and invoked by [Get].
type EyeFactory func(deps Dependencies) Eye

var (
	registryMu sync.RWMutex
	registry   = make(map[string]EyeFactory)
)

// Register adds an eye factory to the global registry.
// Panics if the name is already registered. Intended to be called
// from init() in each eye package.
func Register(name string, factory EyeFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("eye %q already registered", name))
	}

	registry[name] = factory
}

// Get creates a new Eye instance by name, injecting the given dependencies.
// Returns an error if the eye name is not registered.
func Get(name string, deps Dependencies) (Eye, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("unknown eye: %q", name)
	}

	return factory(deps), nil
}

// Exists reports whether an eye with the given name is registered.
func Exists(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	_, exists := registry[name]
	return exists
}

// List returns all registered eye names in sorted order.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}
