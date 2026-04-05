package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadFromYAML(t *testing.T) {
	content := `
default: primary
migration_table: schema_migrations
connections:
  primary:
    driver: postgres
    host: localhost
    port: 5432
    database: testdb
    username: user
    password: pass
  secondary:
    driver: mysql
    host: 127.0.0.1
    port: 3306
    database: otherdb
    username: root
    password: secret
`
	path := writeTestFile(t, "config.yaml", content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultConn != "primary" {
		t.Errorf("got %v, want %v", cfg.DefaultConn, "primary")
	}
	if cfg.MigrationTable != "schema_migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationTable, "schema_migrations")
	}
	if len(cfg.Connections) != 2 {
		t.Errorf("len = %d, want %d", len(cfg.Connections), 2)
	}

	primary := cfg.Connections["primary"]
	if primary.Driver != "postgres" {
		t.Errorf("got %v, want %v", primary.Driver, "postgres")
	}
	if primary.Host != "localhost" {
		t.Errorf("got %v, want %v", primary.Host, "localhost")
	}
	if primary.Port != 5432 {
		t.Errorf("got %v, want %v", primary.Port, 5432)
	}
	if primary.Database != "testdb" {
		t.Errorf("got %v, want %v", primary.Database, "testdb")
	}

	secondary := cfg.Connections["secondary"]
	if secondary.Driver != "mysql" {
		t.Errorf("got %v, want %v", secondary.Driver, "mysql")
	}
	if secondary.Host != "127.0.0.1" {
		t.Errorf("got %v, want %v", secondary.Host, "127.0.0.1")
	}
	if secondary.Port != 3306 {
		t.Errorf("got %v, want %v", secondary.Port, 3306)
	}
}

func TestLoadFromYML(t *testing.T) {
	content := `
connections:
  main:
    driver: sqlite
    host: ":memory:"
    database: test.db
`
	path := writeTestFile(t, "config.yml", content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Connections) != 1 {
		t.Errorf("len = %d, want %d", len(cfg.Connections), 1)
	}
	if cfg.Connections["main"].Driver != "sqlite" {
		t.Errorf("got %v, want %v", cfg.Connections["main"].Driver, "sqlite")
	}
}

func TestLoadFromJSON(t *testing.T) {
	cfgData := Config{
		DefaultConn:    "main",
		MigrationTable: "my_migrations",
		Connections: map[string]ConnectionConfig{
			"main": {
				Driver:   "postgres",
				Host:     "db.example.com",
				Port:     5432,
				Database: "production",
				Username: "admin",
				Password: "secret",
			},
		},
	}

	data, err := json.MarshalIndent(cfgData, "", "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := writeTestFile(t, "config.json", string(data))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultConn != "main" {
		t.Errorf("got %v, want %v", cfg.DefaultConn, "main")
	}
	if cfg.MigrationTable != "my_migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationTable, "my_migrations")
	}
	if cfg.Connections["main"].Driver != "postgres" {
		t.Errorf("got %v, want %v", cfg.Connections["main"].Driver, "postgres")
	}
	if cfg.Connections["main"].Host != "db.example.com" {
		t.Errorf("got %v, want %v", cfg.Connections["main"].Host, "db.example.com")
	}
}

func TestLoadUnsupportedFormat(t *testing.T) {
	path := writeTestFile(t, "config.toml", "key = value")

	_, err := Load(path)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported config file format") {
		t.Errorf("%q not in %q", "unsupported config file format", err.Error())
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("%q not in %q", "failed to read config file", err.Error())
	}
}

func TestLoadFromEnv(t *testing.T) {
	envVars := map[string]string{
		"GOMIGRATE_DEFAULT_CONNECTION": "primary",
		"GOMIGRATE_DB_DRIVER":          "postgres",
		"GOMIGRATE_DB_HOST":            "localhost",
		"GOMIGRATE_DB_PORT":            "5432",
		"GOMIGRATE_DB_DATABASE":        "testdb",
		"GOMIGRATE_DB_USERNAME":        "user",
		"GOMIGRATE_DB_PASSWORD":        "pass",
		"GOMIGRATE_MIGRATION_TABLE":    "schema_migrations",
		"GOMIGRATE_MIGRATION_DIR":      "db/migrations",
		"GOMIGRATE_SEEDER_DIR":         "db/seeders",
		"GOMIGRATE_LOG_LEVEL":          "debug",
		"GOMIGRATE_LOG_OUTPUT":         "file",
	}
	setEnvVars(t, envVars)

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultConn != "primary" {
		t.Errorf("got %v, want %v", cfg.DefaultConn, "primary")
	}
	if cfg.MigrationTable != "schema_migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationTable, "schema_migrations")
	}
	if cfg.MigrationDir != "db/migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationDir, "db/migrations")
	}
	if cfg.SeederDir != "db/seeders" {
		t.Errorf("got %v, want %v", cfg.SeederDir, "db/seeders")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("got %v, want %v", cfg.LogLevel, "debug")
	}
	if cfg.LogOutput != "file" {
		t.Errorf("got %v, want %v", cfg.LogOutput, "file")
	}

	conn := cfg.Connections["primary"]
	if conn.Driver != "postgres" {
		t.Errorf("got %v, want %v", conn.Driver, "postgres")
	}
	if conn.Host != "localhost" {
		t.Errorf("got %v, want %v", conn.Host, "localhost")
	}
	if conn.Port != 5432 {
		t.Errorf("got %v, want %v", conn.Port, 5432)
	}
	if conn.Database != "testdb" {
		t.Errorf("got %v, want %v", conn.Database, "testdb")
	}
	if conn.Username != "user" {
		t.Errorf("got %v, want %v", conn.Username, "user")
	}
	if conn.Password != "pass" {
		t.Errorf("got %v, want %v", conn.Password, "pass")
	}
}

func TestLoadFromEnvInvalidPort(t *testing.T) {
	setEnvVars(t, map[string]string{
		"GOMIGRATE_DB_DRIVER":   "postgres",
		"GOMIGRATE_DB_HOST":     "localhost",
		"GOMIGRATE_DB_PORT":     "not_a_number",
		"GOMIGRATE_DB_DATABASE": "testdb",
	})

	_, err := LoadFromEnv()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid GOMIGRATE_DB_PORT") {
		t.Errorf("%q not in %q", "invalid GOMIGRATE_DB_PORT", err.Error())
	}
}

func TestLoadFromEnvNoDBVars(t *testing.T) {
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MigrationTable != "migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationTable, "migrations")
	}
	if cfg.MigrationDir != "database/migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationDir, "database/migrations")
	}
	if cfg.SeederDir != "database/seeders" {
		t.Errorf("got %v, want %v", cfg.SeederDir, "database/seeders")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("got %v, want %v", cfg.LogLevel, "info")
	}
	if cfg.LogOutput != "console" {
		t.Errorf("got %v, want %v", cfg.LogOutput, "console")
	}
}

func TestValidateSuccess(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:   "postgres",
				Host:     "localhost",
				Database: "testdb",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateNoConnections(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "at least one connection must be defined") {
		t.Errorf("%q not in %q", "at least one connection must be defined", err.Error())
	}
}

func TestValidateMissingFields(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"broken": {
				Username: "user",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "connections.broken.driver is required") {
		t.Errorf("%q not in %q", "connections.broken.driver is required", err.Error())
	}
	if !strings.Contains(err.Error(), "connections.broken.host is required") {
		t.Errorf("%q not in %q", "connections.broken.host is required", err.Error())
	}
	if !strings.Contains(err.Error(), "connections.broken.database is required") {
		t.Errorf("%q not in %q", "connections.broken.database is required", err.Error())
	}
}

func TestValidatePartialMissingFields(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"partial": {
				Driver: "mysql",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "connections.partial.host is required") {
		t.Errorf("%q not in %q", "connections.partial.host is required", err.Error())
	}
	if !strings.Contains(err.Error(), "connections.partial.database is required") {
		t.Errorf("%q not in %q", "connections.partial.database is required", err.Error())
	}
	if strings.Contains(err.Error(), "connections.partial.driver") {
		t.Errorf("%q should not be in %q", "connections.partial.driver", err.Error())
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()

	if cfg.MigrationTable != "migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationTable, "migrations")
	}
	if cfg.MigrationDir != "database/migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationDir, "database/migrations")
	}
	if cfg.SeederDir != "database/seeders" {
		t.Errorf("got %v, want %v", cfg.SeederDir, "database/seeders")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("got %v, want %v", cfg.LogLevel, "info")
	}
	if cfg.LogOutput != "console" {
		t.Errorf("got %v, want %v", cfg.LogOutput, "console")
	}
	if cfg.Connections == nil {
		t.Errorf("expected non-nil")
	}
}

func TestApplyDefaultsPreservesExisting(t *testing.T) {
	cfg := &Config{
		MigrationTable: "custom_migrations",
		MigrationDir:   "db/migrate",
		SeederDir:      "db/seeds",
		LogLevel:       "debug",
		LogOutput:      "file",
	}
	cfg.ApplyDefaults()

	if cfg.MigrationTable != "custom_migrations" {
		t.Errorf("got %v, want %v", cfg.MigrationTable, "custom_migrations")
	}
	if cfg.MigrationDir != "db/migrate" {
		t.Errorf("got %v, want %v", cfg.MigrationDir, "db/migrate")
	}
	if cfg.SeederDir != "db/seeds" {
		t.Errorf("got %v, want %v", cfg.SeederDir, "db/seeds")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("got %v, want %v", cfg.LogLevel, "debug")
	}
	if cfg.LogOutput != "file" {
		t.Errorf("got %v, want %v", cfg.LogOutput, "file")
	}
}

func TestConfigYAMLRoundTrip(t *testing.T) {
	original := &Config{
		DefaultConn:    "primary",
		MigrationTable: "migrations",
		MigrationDir:   "db/migrations",
		SeederDir:      "db/seeders",
		LogLevel:       "info",
		LogOutput:      "console",
		Connections: map[string]ConnectionConfig{
			"primary": {
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				Username: "admin",
				Password: "secret",
			},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := writeTestFile(t, "roundtrip.yaml", string(data))
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if original.DefaultConn != loaded.DefaultConn {
		t.Errorf("got %v, want %v", loaded.DefaultConn, original.DefaultConn)
	}
	if original.MigrationTable != loaded.MigrationTable {
		t.Errorf("got %v, want %v", loaded.MigrationTable, original.MigrationTable)
	}
	if original.Connections["primary"].Driver != loaded.Connections["primary"].Driver {
		t.Errorf("got %v, want %v", loaded.Connections["primary"].Driver, original.Connections["primary"].Driver)
	}
	if original.Connections["primary"].Host != loaded.Connections["primary"].Host {
		t.Errorf("got %v, want %v", loaded.Connections["primary"].Host, original.Connections["primary"].Host)
	}
	if original.Connections["primary"].Port != loaded.Connections["primary"].Port {
		t.Errorf("got %v, want %v", loaded.Connections["primary"].Port, original.Connections["primary"].Port)
	}
	if original.Connections["primary"].Database != loaded.Connections["primary"].Database {
		t.Errorf("got %v, want %v", loaded.Connections["primary"].Database, original.Connections["primary"].Database)
	}
}

func TestConfigJSONRoundTrip(t *testing.T) {
	original := &Config{
		DefaultConn:    "main",
		MigrationTable: "schema_migrations",
		Connections: map[string]ConnectionConfig{
			"main": {
				Driver:   "mysql",
				Host:     "db.example.com",
				Port:     3306,
				Database: "production",
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := writeTestFile(t, "roundtrip.json", string(data))
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if original.DefaultConn != loaded.DefaultConn {
		t.Errorf("got %v, want %v", loaded.DefaultConn, original.DefaultConn)
	}
	if original.MigrationTable != loaded.MigrationTable {
		t.Errorf("got %v, want %v", loaded.MigrationTable, original.MigrationTable)
	}
	if original.Connections["main"].Driver != loaded.Connections["main"].Driver {
		t.Errorf("got %v, want %v", loaded.Connections["main"].Driver, original.Connections["main"].Driver)
	}
	if original.Connections["main"].Host != loaded.Connections["main"].Host {
		t.Errorf("got %v, want %v", loaded.Connections["main"].Host, original.Connections["main"].Host)
	}
}

// --- helpers ---

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return path
}

func setEnvVars(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestValidateInvalidDriver(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:   "oracle",
				Host:     "localhost",
				Database: "testdb",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "connections.default.driver must be one of: postgres, mysql, sqlite") {
		t.Errorf("%q not in %q", "connections.default.driver must be one of: postgres, mysql, sqlite", err.Error())
	}
}

func TestValidateNegativePort(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:   "postgres",
				Host:     "localhost",
				Port:     -1,
				Database: "testdb",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "connections.default.port must be a positive integer") {
		t.Errorf("%q not in %q", "connections.default.port must be a positive integer", err.Error())
	}
}

func TestValidateZeroPortIsValid(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:   "postgres",
				Host:     "localhost",
				Port:     0,
				Database: "testdb",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	cfg := &Config{
		LogLevel: "verbose",
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:   "postgres",
				Host:     "localhost",
				Database: "testdb",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "log_level must be one of: debug, info, warn, error") {
		t.Errorf("%q not in %q", "log_level must be one of: debug, info, warn, error", err.Error())
	}
}

func TestValidateInvalidLogOutput(t *testing.T) {
	cfg := &Config{
		LogOutput: "syslog",
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:   "postgres",
				Host:     "localhost",
				Database: "testdb",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "log_output must be one of: console, file, both") {
		t.Errorf("%q not in %q", "log_output must be one of: console, file, both", err.Error())
	}
}

func TestValidateNegativeMaxOpenConns(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:       "postgres",
				Host:         "localhost",
				Database:     "testdb",
				MaxOpenConns: -5,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "connections.default.max_open_conns must be non-negative") {
		t.Errorf("%q not in %q", "connections.default.max_open_conns must be non-negative", err.Error())
	}
}

func TestValidateNegativeMaxIdleConns(t *testing.T) {
	cfg := &Config{
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:       "postgres",
				Host:         "localhost",
				Database:     "testdb",
				MaxIdleConns: -3,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	if !strings.Contains(err.Error(), "connections.default.max_idle_conns must be non-negative") {
		t.Errorf("%q not in %q", "connections.default.max_idle_conns must be non-negative", err.Error())
	}
}

func TestValidateMultipleViolations(t *testing.T) {
	cfg := &Config{
		LogLevel:  "verbose",
		LogOutput: "syslog",
		Connections: map[string]ConnectionConfig{
			"default": {
				Driver:       "oracle",
				Host:         "localhost",
				Database:     "testdb",
				Port:         -1,
				MaxOpenConns: -5,
				MaxIdleConns: -3,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("got %v, want %v", err, ErrConfigValidation)
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "driver must be one of") {
		t.Errorf("%q not in %q", "driver must be one of", errMsg)
	}
	if !strings.Contains(errMsg, "port must be a positive integer") {
		t.Errorf("%q not in %q", "port must be a positive integer", errMsg)
	}
	if !strings.Contains(errMsg, "max_open_conns must be non-negative") {
		t.Errorf("%q not in %q", "max_open_conns must be non-negative", errMsg)
	}
	if !strings.Contains(errMsg, "max_idle_conns must be non-negative") {
		t.Errorf("%q not in %q", "max_idle_conns must be non-negative", errMsg)
	}
	if !strings.Contains(errMsg, "log_level must be one of") {
		t.Errorf("%q not in %q", "log_level must be one of", errMsg)
	}
	if !strings.Contains(errMsg, "log_output must be one of") {
		t.Errorf("%q not in %q", "log_output must be one of", errMsg)
	}
}

func TestValidateValidLogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		cfg := &Config{
			LogLevel: level,
			Connections: map[string]ConnectionConfig{
				"default": {
					Driver:   "postgres",
					Host:     "localhost",
					Database: "testdb",
				},
			},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("log_level %q should be valid, got error: %v", level, err)
		}
	}
}

func TestValidateValidLogOutputs(t *testing.T) {
	for _, output := range []string{"console", "file", "both"} {
		cfg := &Config{
			LogOutput: output,
			Connections: map[string]ConnectionConfig{
				"default": {
					Driver:   "postgres",
					Host:     "localhost",
					Database: "testdb",
				},
			},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("log_output %q should be valid, got error: %v", output, err)
		}
	}
}

func TestValidateValidDrivers(t *testing.T) {
	for _, driver := range []string{"postgres", "mysql", "sqlite"} {
		cfg := &Config{
			Connections: map[string]ConnectionConfig{
				"default": {
					Driver:   driver,
					Host:     "localhost",
					Database: "testdb",
				},
			},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("driver %q should be valid, got error: %v", driver, err)
		}
	}
}
