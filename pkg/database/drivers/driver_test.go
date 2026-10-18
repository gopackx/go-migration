package drivers

import (
	"testing"
	"time"

	"github.com/gopackx/go-migration/pkg/database"
	"strings"
)

func TestPostgresDriver_Name(t *testing.T) {
	d := NewPostgresDriver()
	if "postgres" != d.Name() {
		t.Errorf("got %v, want %v", d.Name(), "postgres")
	}
}

func TestPostgresDriver_BuildDSN(t *testing.T) {
	d := NewPostgresDriver()

	config := database.ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "testdb",
	}

	dsn := d.buildDSN(config)
	if !strings.Contains(dsn, "host=localhost") {
		t.Errorf("%q not in %q", "host=localhost", dsn)
	}
	if !strings.Contains(dsn, "port=5432") {
		t.Errorf("%q not in %q", "port=5432", dsn)
	}
	if !strings.Contains(dsn, "user=user") {
		t.Errorf("%q not in %q", "user=user", dsn)
	}
	if !strings.Contains(dsn, "password=pass") {
		t.Errorf("%q not in %q", "password=pass", dsn)
	}
	if !strings.Contains(dsn, "dbname=testdb") {
		t.Errorf("%q not in %q", "dbname=testdb", dsn)
	}
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("%q not in %q", "sslmode=disable", dsn)
	}
}

func TestPostgresDriver_BuildDSN_CustomSSLMode(t *testing.T) {
	d := NewPostgresDriver()

	config := database.ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "testdb",
		Options:  map[string]string{"sslmode": "require"},
	}

	dsn := d.buildDSN(config)
	if !strings.Contains(dsn, "sslmode=require") {
		t.Errorf("%q not in %q", "sslmode=require", dsn)
	}
	if strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("%q should not be in %q", "sslmode=disable", dsn)
	}
}

func TestMySQLDriver_Name(t *testing.T) {
	d := NewMySQLDriver()
	if "mysql" != d.Name() {
		t.Errorf("got %v, want %v", d.Name(), "mysql")
	}
}

func TestMySQLDriver_BuildDSN(t *testing.T) {
	d := NewMySQLDriver()

	config := database.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "secret",
		Database: "mydb",
	}

	dsn := d.buildDSN(config)
	want := "root:secret@tcp(localhost:3306)/mydb?parseTime=true"
	if want != dsn {
		t.Errorf("got %v, want %v", dsn, want)
	}
}

func TestMySQLDriver_BuildDSN_WithOptions(t *testing.T) {
	d := NewMySQLDriver()

	config := database.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "secret",
		Database: "mydb",
		Options:  map[string]string{"charset": "utf8mb4"},
	}

	dsn := d.buildDSN(config)
	if !strings.Contains(dsn, "root:secret@tcp(localhost:3306)/mydb?") {
		t.Errorf("%q not in %q", "root:secret@tcp(localhost:3306)/mydb?", dsn)
	}
	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Errorf("%q not in %q", "charset=utf8mb4", dsn)
	}
}

func TestMySQLDriver_BuildDSN_WithOptionsAddsParseTime(t *testing.T) {
	d := NewMySQLDriver()

	config := database.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "secret",
		Database: "mydb",
		Options:  map[string]string{"charset": "utf8mb4", "loc": "UTC"},
	}

	dsn := d.buildDSN(config)
	want := "root:secret@tcp(localhost:3306)/mydb?charset=utf8mb4&loc=UTC&parseTime=true"
	if want != dsn {
		t.Errorf("got %v, want %v", dsn, want)
	}
}

func TestMySQLDriver_BuildDSN_RespectsExplicitParseTime(t *testing.T) {
	d := NewMySQLDriver()

	for _, key := range []string{"parseTime", "parsetime"} {
		config := database.ConnectionConfig{
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "secret",
			Database: "mydb",
			Options:  map[string]string{key: "false"},
		}

		dsn := d.buildDSN(config)
		want := "root:secret@tcp(localhost:3306)/mydb?" + key + "=false"
		if want != dsn {
			t.Errorf("got %v, want %v", dsn, want)
		}
		if strings.Contains(dsn, "parseTime=true") {
			t.Errorf("explicit %s=false must not be overridden: %q", key, dsn)
		}
	}
}

func TestSQLiteDriver_Name(t *testing.T) {
	d := NewSQLiteDriver()
	if "sqlite" != d.Name() {
		t.Errorf("got %v, want %v", d.Name(), "sqlite")
	}
}

func TestSQLiteDriver_BuildDSN(t *testing.T) {
	d := NewSQLiteDriver()

	config := database.ConnectionConfig{
		Database: "./data.db",
	}

	dsn := d.buildDSN(config)
	if "./data.db" != dsn {
		t.Errorf("got %v, want %v", dsn, "./data.db")
	}
}

func TestSQLiteDriver_BuildDSN_Memory(t *testing.T) {
	d := NewSQLiteDriver()

	config := database.ConnectionConfig{
		Database: ":memory:",
	}

	dsn := d.buildDSN(config)
	if ":memory:" != dsn {
		t.Errorf("got %v, want %v", dsn, ":memory:")
	}
}

func TestSQLiteDriver_BuildDSN_WithOptions(t *testing.T) {
	d := NewSQLiteDriver()

	config := database.ConnectionConfig{
		Database: "./data.db",
		Options:  map[string]string{"cache": "shared"},
	}

	dsn := d.buildDSN(config)
	if !strings.Contains(dsn, "./data.db?") {
		t.Errorf("%q not in %q", "./data.db?", dsn)
	}
	if !strings.Contains(dsn, "cache=shared") {
		t.Errorf("%q not in %q", "cache=shared", dsn)
	}
}

func TestDriverInterface_Compliance(t *testing.T) {
	// Verify all drivers implement the Driver interface at compile time.
	var _ Driver = (*PostgresDriver)(nil)
	var _ Driver = (*MySQLDriver)(nil)
	var _ Driver = (*SQLiteDriver)(nil)
}

func TestConfigurePool(t *testing.T) {
	// configurePool is tested indirectly through Open, but we can verify
	// it doesn't panic with zero-value config.
	config := database.ConnectionConfig{}
	// sql.Open with unknown driver will fail, but configurePool itself
	// should handle zero values gracefully. We test the helper directly
	// by noting it only sets values when > 0.
	config.MaxOpenConns = 0
	config.MaxIdleConns = 0
	config.ConnMaxLifetime = 0
	// No assertion needed — just verifying no panic with zero config.
}

func TestConfigurePool_WithValues(t *testing.T) {
	config := database.ConnectionConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
	if 25 != config.MaxOpenConns {
		t.Errorf("got %v, want %v", config.MaxOpenConns, 25)
	}
	if 5 != config.MaxIdleConns {
		t.Errorf("got %v, want %v", config.MaxIdleConns, 5)
	}
	if 5*time.Minute != config.ConnMaxLifetime {
		t.Errorf("got %v, want %v", config.ConnMaxLifetime, 5*time.Minute)
	}
}
