package app

import (
	"flag"
	"testing"
	"time"

	"github.com/gopackx/go-migration/pkg/cli/commands"
	"github.com/gopackx/go-migration/pkg/seeder"
)

func TestLockSettingsFromFlags(t *testing.T) {
	cases := []struct {
		name         string
		withFlags    bool
		args         []string
		wantTimeout  *time.Duration
		wantDisabled bool
		wantOptions  int
	}{
		{"command without lock flags", false, nil, nil, false, 0},
		{"defaults", true, nil, durationPtr(30 * time.Second), false, 1},
		{"custom timeout", true, []string{"--lock-timeout", "2m"}, durationPtr(2 * time.Minute), false, 1},
		{"no lock", true, []string{"--no-lock"}, durationPtr(30 * time.Second), true, 2},
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
			got, err := lockSettingsFromFlags(fs)
			if err != nil {
				t.Fatal(err)
			}
			if (got.timeout == nil) != (c.wantTimeout == nil) ||
				(got.timeout != nil && *got.timeout != *c.wantTimeout) {
				t.Errorf("timeout = %v, want %v", fmtDuration(got.timeout), fmtDuration(c.wantTimeout))
			}
			if got.disabled != c.wantDisabled {
				t.Errorf("disabled = %v, want %v", got.disabled, c.wantDisabled)
			}
			if n := len(got.options()); n != c.wantOptions {
				t.Errorf("len(options()) = %d, want %d", n, c.wantOptions)
			}
		})
	}
}

func TestLockSettingsFromFlags_RejectsNonDurationFlag(t *testing.T) {
	fs := flag.NewFlagSet("cmd", flag.ContinueOnError)
	fs.String(commands.LockTimeoutFlag, "soon", "")
	if _, err := lockSettingsFromFlags(fs); err == nil {
		t.Fatal("expected an error for a non-duration --lock-timeout flag")
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

func durationPtr(d time.Duration) *time.Duration { return &d }

func fmtDuration(d *time.Duration) string {
	if d == nil {
		return "<nil>"
	}
	return d.String()
}
