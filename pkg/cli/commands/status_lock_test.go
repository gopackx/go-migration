package commands

import (
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gopackx/go-migration/pkg/seeder"
)

func statusCtx(statuses []MigrationStatusInfo) func() *CommandContext {
	return func() *CommandContext {
		return &CommandContext{Migrator: stubMigratorFn{
			statusFn: func() ([]MigrationStatusInfo, error) { return statuses, nil },
		}}
	}
}

func TestMigrateStatusCommand_Check(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	mixed := []MigrationStatusInfo{
		{Name: "20240101000000_create_users", Applied: true, Batch: 1, AppliedAt: &now},
		{Name: "20240102000000_create_posts"},
		{Name: "20240103000000_add_bio"},
	}
	allApplied := mixed[:1]

	cases := []struct {
		name     string
		args     []string
		statuses []MigrationStatusInfo
		wantErr  string
	}{
		{"pending without --check", nil, mixed, ""},
		{"pending with --check", []string{"--check"}, mixed, "2 migration(s) pending"},
		{"all applied with --check", []string{"--check"}, allApplied, ""},
		{"nothing registered with --check", []string{"--check"}, nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			reg, cmd := captureRegister()
			RegisterMigrateStatusCommand(reg, statusCtx(c.statuses), out)

			fs := newFlagSet("migrate:status", cmd.setupFlags)
			if err := fs.Parse(c.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.handler(fs.Args(), fs)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, ErrPendingMigrations) {
					t.Fatalf("err = %v, want ErrPendingMigrations", err)
				}
				if err.Error() != c.wantErr {
					t.Errorf("err = %q, want %q", err.Error(), c.wantErr)
				}
			}
			// The table is printed either way.
			if !strings.Contains(out.String(), "Migration") {
				t.Errorf("status table not printed: %q", out.String())
			}
			for _, s := range c.statuses {
				if !strings.Contains(out.String(), s.Name) {
					t.Errorf("%s missing from output %q", s.Name, out.String())
				}
			}
		})
	}
}

// fakeLocker records Lock/release calls.
type fakeLocker struct {
	events  *[]string
	lockErr error
	// beforeRelease, if set, runs when the lock is released.
	beforeRelease func()
}

func (l fakeLocker) Lock() (func() error, error) {
	*l.events = append(*l.events, "lock")
	if l.lockErr != nil {
		return nil, l.lockErr
	}
	return func() error {
		if l.beforeRelease != nil {
			l.beforeRelease()
		}
		*l.events = append(*l.events, "release")
		return nil
	}, nil
}

// recordingSeeder appends "run" / "rollback" to events.
type recordingSeeder struct{ events *[]string }

func (s recordingSeeder) Run(*sql.DB) error {
	*s.events = append(*s.events, "run")
	return nil
}

func (s recordingSeeder) Rollback(*sql.DB) error {
	*s.events = append(*s.events, "rollback")
	return nil
}

func TestSeedCommands_HoldLockAroundWork(t *testing.T) {
	type registerFn func(reg func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)), getCtx func() *CommandContext)

	cases := []struct {
		name     string
		register registerFn
		args     []string
		work     string
	}{
		{"db:seed", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)), g func() *CommandContext) {
			RegisterSeedCommand(r, g)
		}, nil, "run"},
		{"db:seed --class", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)), g func() *CommandContext) {
			RegisterSeedCommand(r, g)
		}, []string{"--class", "users"}, "run"},
		{"db:seed:rollback", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)), g func() *CommandContext) {
			RegisterSeedRollbackCommand(r, g)
		}, []string{"--class", "users"}, "rollback"},
		{"db:seed:truncate", func(r func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet)), g func() *CommandContext) {
			RegisterSeedTruncateCommand(r, g)
		}, []string{"--table", "users"}, "truncate"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var events []string

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			locker := fakeLocker{events: &events}
			if c.work == "truncate" {
				mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 1))
				// The DELETE must have run by the time the lock is released
				// (and not before it was taken: see the lock event first).
				locker.beforeRelease = func() {
					if mock.ExpectationsWereMet() == nil {
						events = append(events, "truncate")
					}
				}
			}

			registry := seeder.NewRegistry()
			if err := registry.Register("users", recordingSeeder{events: &events}); err != nil {
				t.Fatal(err)
			}
			ctx := &CommandContext{
				Seeder: seeder.NewRunner(registry, db, nil),
				Locker: locker,
			}

			reg, cmd := captureRegister()
			c.register(reg, func() *CommandContext { return ctx })
			fs := newFlagSet(cmd.name, cmd.setupFlags)
			if fs.Lookup(LockTimeoutFlag) == nil || fs.Lookup(NoLockFlag) == nil {
				t.Errorf("%s does not define --%s/--%s", cmd.name, LockTimeoutFlag, NoLockFlag)
			}
			if err := fs.Parse(c.args); err != nil {
				t.Fatal(err)
			}
			if err := cmd.handler(fs.Args(), fs); err != nil {
				t.Fatalf("handler: %v", err)
			}
			want := []string{"lock", c.work, "release"}
			if strings.Join(events, ",") != strings.Join(want, ",") {
				t.Errorf("events = %v, want %v", events, want)
			}
		})
	}
}

func TestSeedCommand_LockErrorSkipsWork(t *testing.T) {
	var events []string
	registry := seeder.NewRegistry()
	if err := registry.Register("users", recordingSeeder{events: &events}); err != nil {
		t.Fatal(err)
	}
	lockErr := errors.New("could not acquire migration lock within 30s")
	ctx := &CommandContext{
		Seeder: seeder.NewRunner(registry, nil, nil),
		Locker: fakeLocker{events: &events, lockErr: lockErr},
	}
	reg, cmd := captureRegister()
	RegisterSeedCommand(reg, func() *CommandContext { return ctx })
	fs := newFlagSet("db:seed", cmd.setupFlags)
	if err := cmd.handler(nil, fs); !errors.Is(err, lockErr) {
		t.Fatalf("err = %v, want lock error", err)
	}
	if strings.Join(events, ",") != "lock" {
		t.Errorf("events = %v, want only the lock attempt", events)
	}
}

func TestLockFlags_OnWritingCommandsOnly(t *testing.T) {
	getCtx := func() *CommandContext { return nil }
	out, in := &bytes.Buffer{}, &bytes.Buffer{}
	type reg = func(string, string, func([]string, *flag.FlagSet) error, func(*flag.FlagSet))
	cases := []struct {
		name     string
		register func(reg)
		wantLock bool
	}{
		{"migrate", func(r reg) { RegisterMigrateCommand(r, getCtx, out) }, true},
		{"migrate:rollback", func(r reg) { RegisterMigrateRollbackCommand(r, getCtx, out) }, true},
		{"migrate:reset", func(r reg) { RegisterMigrateResetCommand(r, getCtx, out, in) }, true},
		{"migrate:refresh", func(r reg) { RegisterMigrateRefreshCommand(r, getCtx, out) }, true},
		{"migrate:fresh", func(r reg) { RegisterMigrateFreshCommand(r, getCtx, out, in) }, true},
		{"migrate:status", func(r reg) { RegisterMigrateStatusCommand(r, getCtx, out) }, false},
		{"migrate:install", func(r reg) { RegisterMigrateInstallCommand(r, getCtx) }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, cmd := captureRegister()
			c.register(r)
			fs := newFlagSet(c.name, cmd.setupFlags)
			timeout := fs.Lookup(LockTimeoutFlag)
			if (timeout != nil) != c.wantLock || (fs.Lookup(NoLockFlag) != nil) != c.wantLock {
				t.Fatalf("lock flags defined = %v, want %v", timeout != nil, c.wantLock)
			}
			if c.wantLock && timeout.DefValue != "30s" {
				t.Errorf("--%s default = %s, want 30s", LockTimeoutFlag, timeout.DefValue)
			}
		})
	}
}
