package database

import (
	"database/sql"
	"fmt"
	"sync"
)

// Driver defines the contract for a database driver that can open connections.
// This is a local alias to avoid requiring callers to import the drivers sub-package
// when working with the Manager directly.
type Driver interface {
	Open(config ConnectionConfig) (*sql.DB, error)
	Name() string
}

// Manager manages multiple named database connections with lazy initialization
// and connection pool configuration.
type Manager struct {
	mu          sync.RWMutex
	connections map[string]*sql.DB
	configs     map[string]ConnectionConfig
	drivers     map[string]Driver
	defaultName string
}

// NewManager creates a new connection Manager with empty registries.
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*sql.DB),
		configs:     make(map[string]ConnectionConfig),
		drivers:     make(map[string]Driver),
	}
}

// RegisterDriver registers a database driver by name (e.g. "postgres", "mysql", "sqlite3").
func (m *Manager) RegisterDriver(name string, driver Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[name] = driver
}

// AddConnection stores a named connection configuration. The actual database
// connection is opened lazily on the first call to Connection().
// If this is the first connection added, it becomes the default.
func (m *Manager) AddConnection(name string, config ConnectionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Driver == "" {
		return fmt.Errorf("connection %q: driver is required", name)
	}

	m.configs[name] = config

	// First connection added becomes the default automatically.
	if m.defaultName == "" {
		m.defaultName = name
	}

	return nil
}

// Connection returns an active database connection for the given name.
// Connections are opened lazily on first request and cached for reuse.
func (m *Manager) Connection(name string) (*sql.DB, error) {
	m.mu.RLock()
	if db, ok := m.connections[name]; ok {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	// Upgrade to write lock to open the connection.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if db, ok := m.connections[name]; ok {
		return db, nil
	}

	config, ok := m.configs[name]
	if !ok {
		return nil, fmt.Errorf("connection %q: %w", name, ErrConnectionNotFound)
	}

	driver, ok := m.drivers[config.Driver]
	if !ok {
		return nil, fmt.Errorf("connection %q: driver %q: %w", name, config.Driver, ErrDriverNotFound)
	}

	db, err := driver.Open(config)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w: %v", name, ErrConnectionFailed, err)
	}

	// Apply pool configuration on top of whatever the driver set.
	applyPoolConfig(db, config)

	m.connections[name] = db
	return db, nil
}

// Default returns the default database connection.
func (m *Manager) Default() (*sql.DB, error) {
	m.mu.RLock()
	name := m.defaultName
	m.mu.RUnlock()

	if name == "" {
		return nil, ErrNoDefault
	}
	return m.Connection(name)
}

// SetDefault sets the default connection name. The named connection must
// already be registered via AddConnection.
func (m *Manager) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.configs[name]; !ok {
		return fmt.Errorf("connection %q: %w", name, ErrConnectionNotFound)
	}
	m.defaultName = name
	return nil
}

// Close closes all open database connections and releases pool resources.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, db := range m.connections {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("closing connection %q: %w", name, err)
		}
	}

	// Clear the connections map so they can't be reused after close.
	m.connections = make(map[string]*sql.DB)

	return firstErr
}
