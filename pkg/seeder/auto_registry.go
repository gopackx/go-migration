package seeder

import (
	"fmt"
	"strings"
	"sync"
)

// seederAutoRegistry stores seeders registered via init() functions.
type seederAutoRegistry struct {
	mu      sync.Mutex
	seeders map[string]Seeder
}

// defaultAutoRegistry is the package-level global auto-registry.
var defaultAutoRegistry = &seederAutoRegistry{
	seeders: make(map[string]Seeder),
}

// AutoRegister registers a seeder in the global auto-registry.
// Intended to be called from init() functions in seeder files.
// Panics if the name is empty/whitespace-only or duplicate (fail-fast at startup).
func AutoRegister(name string, s Seeder) {
	defaultAutoRegistry.mu.Lock()
	defer defaultAutoRegistry.mu.Unlock()

	if strings.TrimSpace(name) == "" {
		panic(fmt.Sprintf("AutoRegister: seeder name %q is invalid", name))
	}
	if _, exists := defaultAutoRegistry.seeders[name]; exists {
		panic(fmt.Sprintf("AutoRegister: duplicate seeder name %q", name))
	}
	defaultAutoRegistry.seeders[name] = s
}

// GetAutoRegistered returns a defensive copy of all auto-registered seeders.
func GetAutoRegistered() map[string]Seeder {
	defaultAutoRegistry.mu.Lock()
	defer defaultAutoRegistry.mu.Unlock()

	result := make(map[string]Seeder, len(defaultAutoRegistry.seeders))
	for k, v := range defaultAutoRegistry.seeders {
		result[k] = v
	}
	return result
}

// ResetAutoRegistry clears the global auto-registry (for testing only).
func ResetAutoRegistry() {
	defaultAutoRegistry.mu.Lock()
	defer defaultAutoRegistry.mu.Unlock()

	defaultAutoRegistry.seeders = make(map[string]Seeder)
}
