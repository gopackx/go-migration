package migrator

import (
	"flag"
	"testing"
	"time"

	"github.com/gopackx/go-migration/pkg/cli/commands"
	"github.com/gopackx/go-migration/pkg/seeder"
)

func TestLockOptions(t *testing.T) {
	cases := []struct {
		name         string
		withFlags    bool
		args         []string
		wantTimeout  time.Duration
		wantDisabled bool
	}{
		{"command without lock flags", false, nil, DefaultLockTimeout, false},
		{"defaults", true, nil, 30 * time.Second, false},
		{"custom timeout", true, []string{"--lock-timeout", "2m"}, 2 * time.Minute, false},
		{"no lock", true, []string{"--no-lock"}, 30 * time.Second, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := flag.NewFlagSet("cmd", flag.ContinueOnError)
			if c.withFlags {
				commands.AddLockFlags(fs)
			}
			if err := fs.Parse(c.args); err != nil {
				t.Fatal(err)
			}
			opts, err := lockOptions(fs)
			if err != nil {
				t.Fatal(err)
			}
			m := New(nil, opts...)
			if m.lockTimeout != c.wantTimeout || m.lockDisabled != c.wantDisabled {
				t.Errorf("lockTimeout=%v lockDisabled=%v, want %v %v", m.lockTimeout, m.lockDisabled, c.wantTimeout, c.wantDisabled)
			}
		})
	}
	if commands.DefaultLockTimeout != DefaultLockTimeout {
		t.Errorf("commands.DefaultLockTimeout = %v, migrator.DefaultLockTimeout = %v", commands.DefaultLockTimeout, DefaultLockTimeout)
	}
}

func TestSeederDialect(t *testing.T) {
	for driver, want := range map[string]seeder.Dialect{
		"postgres": seeder.DialectPostgres,
		"mysql":    seeder.DialectMySQL,
		"sqlite":   seeder.DialectSQLite,
		"sqlite3":  seeder.DialectSQLite,
	} {
		if got := seederDialect(driver); got != want {
			t.Errorf("seederDialect(%q) = %v, want %v", driver, got, want)
		}
	}
}
