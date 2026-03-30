package eyes

import (
	"fmt"
	"sort"
	"sync"
)

// EyeFactory creates a new instance of an Eye.
type EyeFactory func() Eye

var (
	registryMu sync.RWMutex
	registry   = make(map[string]EyeFactory)
)

// Register adds an eye factory to the global registry.
// Panics if the name is already registered.
func Register(name string, factory EyeFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("eye %q already registered", name))
	}

	registry[name] = factory
}

// Get returns a new Eye instance by name.
func Get(name string) (Eye, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("unknown eye: %q", name)
	}

	return factory(), nil
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
