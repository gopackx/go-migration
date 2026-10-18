// Package config provides configuration loading and validation for go-migration.
// It supports YAML files, JSON files, and environment variables.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrConfigValidation is returned when configuration validation fails.
var ErrConfigValidation = errors.New("configuration validation failed")

// Config holds the top-level configuration for go-migration.
type Config struct {
	Connections    map[string]ConnectionConfig `yaml:"connections" json:"connections"`
	DefaultConn    string                      `yaml:"default" json:"default"`
	MigrationTable string                      `yaml:"migration_table" json:"migration_table"`
	MigrationDir   string                      `yaml:"migration_dir" json:"migration_dir"`
	SeederDir      string                      `yaml:"seeder_dir" json:"seeder_dir"`
	FactoryDir     string                      `yaml:"factory_dir" json:"factory_dir"`
	LogLevel       string                      `yaml:"log_level" json:"log_level"`
	LogOutput      string                      `yaml:"log_output" json:"log_output"`
}

// ConnectionConfig holds the configuration for a single database connection.
type ConnectionConfig struct {
	Driver          string            `yaml:"driver" json:"driver"`
	Host            string            `yaml:"host" json:"host"`
	Port            int               `yaml:"port" json:"port"`
	Database        string            `yaml:"database" json:"database"`
	Username        string            `yaml:"username" json:"username"`
	Password        string            `yaml:"password" json:"password"`
	MaxOpenConns    int               `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int               `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration     `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	Options         map[string]string `yaml:"options" json:"options"`
}

// ApplyDefaults sets sensible default values for optional settings.
func (c *Config) ApplyDefaults() {
	if c.MigrationTable == "" {
		c.MigrationTable = "migrations"
	}
	if c.MigrationDir == "" {
		c.MigrationDir = "database/migrations"
	}
	if c.SeederDir == "" {
		c.SeederDir = "database/seeders"
	}
	if c.FactoryDir == "" {
		c.FactoryDir = "database/factories"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.LogOutput == "" {
		c.LogOutput = "console"
	}
	if c.Connections == nil {
		c.Connections = make(map[string]ConnectionConfig)
	}
}

// isSQLiteDriver reports whether driver names the SQLite driver.
func isSQLiteDriver(driver string) bool {
	return driver == "sqlite" || driver == "sqlite3"
}

// Validate checks that the configuration has all required fields and valid values.
// Returns an error wrapping ErrConfigValidation listing all violations.
func (c *Config) Validate() error {
	var violations []string

	if len(c.Connections) == 0 {
		violations = append(violations, "at least one connection must be defined")
	}

	// The SQLite driver (modernc.org/sqlite) registers itself with
	// database/sql as "sqlite"; "sqlite3" is accepted as an alias for
	// configurations written for other SQLite drivers.
	validDrivers := map[string]bool{"postgres": true, "mysql": true, "sqlite": true, "sqlite3": true}
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	validLogOutputs := map[string]bool{"console": true, "file": true, "both": true}

	for name, conn := range c.Connections {
		if conn.Driver == "" {
			violations = append(violations, fmt.Sprintf("connections.%s.driver is required", name))
		} else if !validDrivers[conn.Driver] {
			violations = append(violations, fmt.Sprintf("connections.%s.driver must be one of: postgres, mysql, sqlite, sqlite3", name))
		}
		// SQLite is file based: the database path is all it needs.
		if conn.Host == "" && !isSQLiteDriver(conn.Driver) {
			violations = append(violations, fmt.Sprintf("connections.%s.host is required", name))
		}
		if conn.Database == "" {
			violations = append(violations, fmt.Sprintf("connections.%s.database is required", name))
		}
		if conn.Port != 0 && conn.Port < 0 {
			violations = append(violations, fmt.Sprintf("connections.%s.port must be a positive integer", name))
		}
		if conn.MaxOpenConns < 0 {
			violations = append(violations, fmt.Sprintf("connections.%s.max_open_conns must be non-negative", name))
		}
		if conn.MaxIdleConns < 0 {
			violations = append(violations, fmt.Sprintf("connections.%s.max_idle_conns must be non-negative", name))
		}
	}

	if c.LogLevel != "" && !validLogLevels[c.LogLevel] {
		violations = append(violations, "log_level must be one of: debug, info, warn, error")
	}
	if c.LogOutput != "" && !validLogOutputs[c.LogOutput] {
		violations = append(violations, "log_output must be one of: console, file, both")
	}

	if len(violations) > 0 {
		return fmt.Errorf("%w: %s", ErrConfigValidation, strings.Join(violations, ", "))
	}
	return nil
}

// Load reads configuration from a file at the given path.
// It detects the format by file extension (.yaml/.yml for YAML, .json for JSON).
// After loading, it applies defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	data, err = InterpolateEnv(data)
	if err != nil {
		return nil, fmt.Errorf("failed to interpolate env variables in config: %w", err)
	}

	cfg := &Config{}
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		fmt.Fprintf(os.Stderr, "WARNING: YAML configuration format is deprecated and will be removed in a future version. Please migrate to JSON format.\n")
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	cfg.ApplyDefaults()
	return cfg, nil
}

// LoadFromEnv builds a Config from environment variables with the GOMIGRATE_ prefix.
// It creates a single "default" connection from the DB-related env vars.
// After loading, it applies defaults.
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		Connections: make(map[string]ConnectionConfig),
	}

	// Top-level settings
	cfg.DefaultConn = getEnv("GOMIGRATE_DEFAULT_CONNECTION", "default")
	cfg.MigrationTable = getEnv("GOMIGRATE_MIGRATION_TABLE", "")
	cfg.MigrationDir = getEnv("GOMIGRATE_MIGRATION_DIR", "")
	cfg.SeederDir = getEnv("GOMIGRATE_SEEDER_DIR", "")
	cfg.FactoryDir = getEnv("GOMIGRATE_FACTORY_DIR", "")
	cfg.LogLevel = getEnv("GOMIGRATE_LOG_LEVEL", "")
	cfg.LogOutput = getEnv("GOMIGRATE_LOG_OUTPUT", "")

	// Build default connection from env vars
	conn := ConnectionConfig{
		Driver:   getEnv("GOMIGRATE_DB_DRIVER", ""),
		Host:     getEnv("GOMIGRATE_DB_HOST", ""),
		Database: getEnv("GOMIGRATE_DB_DATABASE", ""),
		Username: getEnv("GOMIGRATE_DB_USERNAME", ""),
		Password: getEnv("GOMIGRATE_DB_PASSWORD", ""),
	}

	if portStr := getEnv("GOMIGRATE_DB_PORT", ""); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid GOMIGRATE_DB_PORT value %q: %w", portStr, err)
		}
		conn.Port = port
	}

	// Only add the connection if at least the driver is specified
	if conn.Driver != "" || conn.Host != "" || conn.Database != "" {
		connName := cfg.DefaultConn
		cfg.Connections[connName] = conn
	}

	cfg.ApplyDefaults()
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
