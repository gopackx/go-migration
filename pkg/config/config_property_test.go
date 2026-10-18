package config

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

// Valid value sets for config fields.
var (
	validDrivers    = []string{"postgres", "mysql", "sqlite", "sqlite3"}
	validLogLevels  = []string{"debug", "info", "warn", "error"}
	validLogOutputs = []string{"console", "file", "both"}
)

// genValidDriver generates a valid driver string.
func genValidDriver() *rapid.Generator[string] {
	return rapid.SampledFrom(validDrivers)
}

// genInvalidDriver generates a driver string NOT in the valid set.
func genInvalidDriver() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		s := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "invalidDriver")
		for s == "postgres" || s == "mysql" || s == "sqlite" || s == "sqlite3" || s == "" {
			s = rapid.StringMatching(`[a-z]{2,10}`).Draw(t, "invalidDriver")
		}
		return s
	})
}

// genValidLogLevel generates a valid log_level string (including empty, which is valid).
func genValidLogLevel() *rapid.Generator[string] {
	return rapid.SampledFrom(append([]string{""}, validLogLevels...))
}

// genInvalidLogLevel generates a log_level string NOT in the valid set and non-empty.
func genInvalidLogLevel() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		s := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "invalidLogLevel")
		for s == "debug" || s == "info" || s == "warn" || s == "error" || s == "" {
			s = rapid.StringMatching(`[a-z]{2,10}`).Draw(t, "invalidLogLevel")
		}
		return s
	})
}

// genValidLogOutput generates a valid log_output string (including empty).
func genValidLogOutput() *rapid.Generator[string] {
	return rapid.SampledFrom(append([]string{""}, validLogOutputs...))
}

// genInvalidLogOutput generates a log_output string NOT in the valid set and non-empty.
func genInvalidLogOutput() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		s := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "invalidLogOutput")
		for s == "console" || s == "file" || s == "both" || s == "" {
			s = rapid.StringMatching(`[a-z]{2,10}`).Draw(t, "invalidLogOutput")
		}
		return s
	})
}

// genValidPort generates a valid port (0 or positive).
func genValidPort() *rapid.Generator[int] {
	return rapid.IntRange(0, 65535)
}

// genNegativeInt generates a negative integer.
func genNegativeInt() *rapid.Generator[int] {
	return rapid.IntRange(-10000, -1)
}

// genNonNegativeInt generates a non-negative integer.
func genNonNegativeInt() *rapid.Generator[int] {
	return rapid.IntRange(0, 10000)
}

// genConnName generates a simple connection name.
func genConnName() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{1,8}`)
}

// genHost generates a non-empty host string.
func genHost() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{1,10}\.example\.com`)
}

// genDatabase generates a non-empty database name.
func genDatabase() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{1,10}db`)
}

// genFullyValidConfig generates a Config where all fields have valid values.
func genFullyValidConfig() *rapid.Generator[*Config] {
	return rapid.Custom(func(t *rapid.T) *Config {
		connName := genConnName().Draw(t, "connName")
		return &Config{
			LogLevel:  genValidLogLevel().Draw(t, "logLevel"),
			LogOutput: genValidLogOutput().Draw(t, "logOutput"),
			Connections: map[string]ConnectionConfig{
				connName: {
					Driver:       genValidDriver().Draw(t, "driver"),
					Host:         genHost().Draw(t, "host"),
					Database:     genDatabase().Draw(t, "database"),
					Port:         genValidPort().Draw(t, "port"),
					MaxOpenConns: genNonNegativeInt().Draw(t, "maxOpen"),
					MaxIdleConns: genNonNegativeInt().Draw(t, "maxIdle"),
				},
			},
		}
	})
}

// Feature: library-improvements, Property 16: Config field validation rejects invalid values
// **Validates: Requirements 9.1, 9.2, 9.3, 9.4, 9.6**

func TestProperty16_ConfigFieldValidationRejectsInvalidValues(t *testing.T) {
	// Sub-test: fully valid configs pass validation.
	t.Run("valid_configs_pass", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			err := cfg.Validate()
			if err != nil {
				t.Errorf("a fully valid config should pass validation")
			}
		})
	})

	// Sub-test: invalid driver causes validation failure.
	t.Run("invalid_driver_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			// Replace the driver with an invalid one in the first connection.
			for name, conn := range cfg.Connections {
				conn.Driver = genInvalidDriver().Draw(t, "badDriver")
				cfg.Connections[name] = conn
				break
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("invalid driver should cause validation failure")
			}
			if !strings.Contains(err.Error(), "driver must be one of") {
				t.Errorf("%q not in %q", "driver must be one of", err.Error())
			}
		})
	})

	// Sub-test: negative port causes validation failure.
	t.Run("negative_port_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			for name, conn := range cfg.Connections {
				conn.Port = genNegativeInt().Draw(t, "badPort")
				cfg.Connections[name] = conn
				break
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("negative port should cause validation failure")
			}
			if !strings.Contains(err.Error(), "port must be a positive integer") {
				t.Errorf("%q not in %q", "port must be a positive integer", err.Error())
			}
		})
	})

	// Sub-test: invalid log_level causes validation failure.
	t.Run("invalid_log_level_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			cfg.LogLevel = genInvalidLogLevel().Draw(t, "badLogLevel")
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("invalid log_level should cause validation failure")
			}
			if !strings.Contains(err.Error(), "log_level must be one of") {
				t.Errorf("%q not in %q", "log_level must be one of", err.Error())
			}
		})
	})

	// Sub-test: invalid log_output causes validation failure.
	t.Run("invalid_log_output_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			cfg.LogOutput = genInvalidLogOutput().Draw(t, "badLogOutput")
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("invalid log_output should cause validation failure")
			}
			if !strings.Contains(err.Error(), "log_output must be one of") {
				t.Errorf("%q not in %q", "log_output must be one of", err.Error())
			}
		})
	})

	// Sub-test: negative max_open_conns causes validation failure.
	t.Run("negative_max_open_conns_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			for name, conn := range cfg.Connections {
				conn.MaxOpenConns = genNegativeInt().Draw(t, "badMaxOpen")
				cfg.Connections[name] = conn
				break
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("negative max_open_conns should cause validation failure")
			}
			if !strings.Contains(err.Error(), "max_open_conns must be non-negative") {
				t.Errorf("%q not in %q", "max_open_conns must be non-negative", err.Error())
			}
		})
	})

	// Sub-test: negative max_idle_conns causes validation failure.
	t.Run("negative_max_idle_conns_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := genFullyValidConfig().Draw(t, "cfg")
			for name, conn := range cfg.Connections {
				conn.MaxIdleConns = genNegativeInt().Draw(t, "badMaxIdle")
				cfg.Connections[name] = conn
				break
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("negative max_idle_conns should cause validation failure")
			}
			if !strings.Contains(err.Error(), "max_idle_conns must be non-negative") {
				t.Errorf("%q not in %q", "max_idle_conns must be non-negative", err.Error())
			}
		})
	})
}

// Feature: library-improvements, Property 17: All validation failures reported
// **Validates: Requirements 9.5**

func TestProperty17_AllValidationFailuresReported(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		connName := genConnName().Draw(t, "connName")

		// Start with a valid config, then inject multiple violations.
		conn := ConnectionConfig{
			Driver:       genValidDriver().Draw(t, "driver"),
			Host:         genHost().Draw(t, "host"),
			Database:     genDatabase().Draw(t, "database"),
			Port:         genValidPort().Draw(t, "port"),
			MaxOpenConns: genNonNegativeInt().Draw(t, "maxOpen"),
			MaxIdleConns: genNonNegativeInt().Draw(t, "maxIdle"),
		}
		cfg := &Config{
			LogLevel:    genValidLogLevel().Draw(t, "logLevel"),
			LogOutput:   genValidLogOutput().Draw(t, "logOutput"),
			Connections: map[string]ConnectionConfig{connName: conn},
		}

		// Track which violations we inject.
		type violation struct {
			name    string
			snippet string // substring expected in error message
		}
		var injected []violation

		// Randomly decide which violations to inject (at least 2).
		injectInvalidDriver := rapid.Bool().Draw(t, "injectDriver")
		injectNegativePort := rapid.Bool().Draw(t, "injectPort")
		injectInvalidLogLevel := rapid.Bool().Draw(t, "injectLogLevel")
		injectInvalidLogOutput := rapid.Bool().Draw(t, "injectLogOutput")
		injectNegMaxOpen := rapid.Bool().Draw(t, "injectMaxOpen")
		injectNegMaxIdle := rapid.Bool().Draw(t, "injectMaxIdle")

		if injectInvalidDriver {
			conn.Driver = genInvalidDriver().Draw(t, "badDriver")
			injected = append(injected, violation{"driver", "driver must be one of"})
		}
		if injectNegativePort {
			conn.Port = genNegativeInt().Draw(t, "badPort")
			injected = append(injected, violation{"port", "port must be a positive integer"})
		}
		if injectInvalidLogLevel {
			cfg.LogLevel = genInvalidLogLevel().Draw(t, "badLogLevel")
			injected = append(injected, violation{"log_level", "log_level must be one of"})
		}
		if injectInvalidLogOutput {
			cfg.LogOutput = genInvalidLogOutput().Draw(t, "badLogOutput")
			injected = append(injected, violation{"log_output", "log_output must be one of"})
		}
		if injectNegMaxOpen {
			conn.MaxOpenConns = genNegativeInt().Draw(t, "badMaxOpen")
			injected = append(injected, violation{"max_open_conns", "max_open_conns must be non-negative"})
		}
		if injectNegMaxIdle {
			conn.MaxIdleConns = genNegativeInt().Draw(t, "badMaxIdle")
			injected = append(injected, violation{"max_idle_conns", "max_idle_conns must be non-negative"})
		}

		// We need at least 2 violations for this property.
		if len(injected) < 2 {
			// Force two violations if random booleans didn't give us enough.
			conn.Driver = genInvalidDriver().Draw(t, "forcedBadDriver")
			injected = append(injected, violation{"driver", "driver must be one of"})
			cfg.LogLevel = genInvalidLogLevel().Draw(t, "forcedBadLogLevel")
			injected = append(injected, violation{"log_level", "log_level must be one of"})
		}

		// Update the connection in the config.
		cfg.Connections[connName] = conn

		err := cfg.Validate()
		if err == nil {
			t.Fatalf("config with %d violations should fail validation", len(injected))
		}

		errMsg := err.Error()

		// Verify ALL injected violations are referenced in the error.
		for _, v := range injected {
			if !strings.Contains(errMsg, v.snippet) {
				t.Errorf("error should reference %s violation (expected substring %q in %q)", v.name, v.snippet, errMsg)
			}
		}
	})
}

// Feature: reduce-external-dependencies, Property 5: Config JSON Round-Trip
// For any valid Config, JSON marshal/unmarshal produces an equivalent object.
func TestProperty5_ConfigJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := genFullyValidConfig().Draw(t, "cfg")
		cfg.ApplyDefaults()

		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var restored Config
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}
		restored.ApplyDefaults()

		if cfg.DefaultConn != restored.DefaultConn {
			t.Errorf("DefaultConn: got %q, want %q", restored.DefaultConn, cfg.DefaultConn)
		}
		if cfg.MigrationTable != restored.MigrationTable {
			t.Errorf("MigrationTable: got %q, want %q", restored.MigrationTable, cfg.MigrationTable)
		}
		if cfg.MigrationDir != restored.MigrationDir {
			t.Errorf("MigrationDir: got %q, want %q", restored.MigrationDir, cfg.MigrationDir)
		}
		if cfg.LogLevel != restored.LogLevel {
			t.Errorf("LogLevel: got %q, want %q", restored.LogLevel, cfg.LogLevel)
		}
		if cfg.LogOutput != restored.LogOutput {
			t.Errorf("LogOutput: got %q, want %q", restored.LogOutput, cfg.LogOutput)
		}
		if len(cfg.Connections) != len(restored.Connections) {
			t.Fatalf("Connections count: got %d, want %d", len(restored.Connections), len(cfg.Connections))
		}
		for name, origConn := range cfg.Connections {
			resConn, ok := restored.Connections[name]
			if !ok {
				t.Fatalf("connection %q missing after round-trip", name)
			}
			if origConn.Driver != resConn.Driver {
				t.Errorf("conn %s Driver: got %q, want %q", name, resConn.Driver, origConn.Driver)
			}
			if origConn.Host != resConn.Host {
				t.Errorf("conn %s Host: got %q, want %q", name, resConn.Host, origConn.Host)
			}
			if origConn.Port != resConn.Port {
				t.Errorf("conn %s Port: got %d, want %d", name, resConn.Port, origConn.Port)
			}
			if origConn.Database != resConn.Database {
				t.Errorf("conn %s Database: got %q, want %q", name, resConn.Database, origConn.Database)
			}
		}
	})
}

// Feature: reduce-external-dependencies, Property 6: Cross-Format Config Equivalence
// For any valid Config, parsing from JSON and YAML representations produces equivalent objects.
func TestProperty6_CrossFormatConfigEquivalence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := genFullyValidConfig().Draw(t, "cfg")
		cfg.ApplyDefaults()

		// Marshal to JSON
		jsonData, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		// Marshal to YAML
		yamlData, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("yaml.Marshal failed: %v", err)
		}

		// Unmarshal both
		var fromJSON Config
		if err := json.Unmarshal(jsonData, &fromJSON); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}
		fromJSON.ApplyDefaults()

		var fromYAML Config
		if err := yaml.Unmarshal(yamlData, &fromYAML); err != nil {
			t.Fatalf("yaml.Unmarshal failed: %v", err)
		}
		fromYAML.ApplyDefaults()

		// Compare the two loaded configs
		if fromJSON.DefaultConn != fromYAML.DefaultConn {
			t.Errorf("DefaultConn: JSON=%q, YAML=%q", fromJSON.DefaultConn, fromYAML.DefaultConn)
		}
		if fromJSON.MigrationTable != fromYAML.MigrationTable {
			t.Errorf("MigrationTable: JSON=%q, YAML=%q", fromJSON.MigrationTable, fromYAML.MigrationTable)
		}
		if fromJSON.MigrationDir != fromYAML.MigrationDir {
			t.Errorf("MigrationDir: JSON=%q, YAML=%q", fromJSON.MigrationDir, fromYAML.MigrationDir)
		}
		if fromJSON.LogLevel != fromYAML.LogLevel {
			t.Errorf("LogLevel: JSON=%q, YAML=%q", fromJSON.LogLevel, fromYAML.LogLevel)
		}
		if len(fromJSON.Connections) != len(fromYAML.Connections) {
			t.Fatalf("Connections count: JSON=%d, YAML=%d", len(fromJSON.Connections), len(fromYAML.Connections))
		}
		for name, jConn := range fromJSON.Connections {
			yConn, ok := fromYAML.Connections[name]
			if !ok {
				t.Fatalf("connection %q in JSON but not in YAML", name)
			}
			if jConn.Driver != yConn.Driver {
				t.Errorf("conn %s Driver: JSON=%q, YAML=%q", name, jConn.Driver, yConn.Driver)
			}
			if jConn.Host != yConn.Host {
				t.Errorf("conn %s Host: JSON=%q, YAML=%q", name, jConn.Host, yConn.Host)
			}
			if jConn.Port != yConn.Port {
				t.Errorf("conn %s Port: JSON=%d, YAML=%d", name, jConn.Port, yConn.Port)
			}
			if jConn.Database != yConn.Database {
				t.Errorf("conn %s Database: JSON=%q, YAML=%q", name, jConn.Database, yConn.Database)
			}
		}
	})
}
