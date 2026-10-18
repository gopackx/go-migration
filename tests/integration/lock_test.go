//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
)

const insertAdmin = "20240101000010_insert_admin_user"

// countedMigration counts how many times its Up runs (across Migrators in
// this process) and can be slowed down to widen the window in which two
// concurrent Up calls overlap.
type countedMigration struct {
	inner migrator.Migration
	runs  *atomic.Int32
	delay time.Duration
}

func (m countedMigration) Up(s *schema.Builder) error {
	m.runs.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.inner.Up(s)
}

func (m countedMigration) Down(s *schema.Builder) error { return m.inner.Down(s) }

// insertAdminMigration is a data migration. Without the migration lock, two
// processes could both run it before either records it.
func insertAdminMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			return s.RawExec("INSERT INTO users (name, email) VALUES ('admin', 'admin@example.com')")
		},
		down: func(s *schema.Builder) error {
			return s.RawExec("DELETE FROM users WHERE email = 'admin@example.com'")
		},
	}
}

func requireLockingBackend(t *testing.T, b backend) {
	t.Helper()
	if b.openPeer == nil {
		t.Skip("SQLite uses no migration lock: its database file lock serialises writers")
	}
}

// TestConcurrentUpAppliesEachMigrationOnce starts two Migrators on separate
// connection pools (standing in for two processes, e.g. two Kubernetes Job
// pods) at the same moment. Both must succeed, and every migration must run
// exactly once and be recorded exactly once.
func TestConcurrentUpAppliesEachMigrationOnce(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		requireLockingBackend(t, b)
		peer := b.openPeer(t)

		names := []string{createUsers, createPosts, addBio, insertAdmin}
		runs := make(map[string]*atomic.Int32, len(names))
		for _, n := range names {
			runs[n] = new(atomic.Int32)
		}
		build := func(conn *sql.DB) *migrator.Migrator {
			m := b.newMigrator(conn)
			for _, n := range names {
				var inner migrator.Migration
				var delay time.Duration
				switch n {
				case createUsers:
					inner, delay = createUsersMigration(), 300*time.Millisecond
				case createPosts:
					inner = createPostsMigration()
				case addBio:
					inner = addBioMigration()
				case insertAdmin:
					inner = insertAdminMigration()
				}
				must(t, m.Register(n, countedMigration{inner: inner, runs: runs[n], delay: delay}), "register "+n)
			}
			return m
		}
		migrators := []*migrator.Migrator{build(db), build(peer)}

		start := make(chan struct{})
		errs := make([]error, len(migrators))
		var wg sync.WaitGroup
		for i, m := range migrators {
			wg.Add(1)
			go func(i int, m *migrator.Migrator) {
				defer wg.Done()
				<-start
				errs[i] = m.Up()
			}(i, m)
		}
		close(start)
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("Up #%d: %v", i, err)
			}
		}
		for _, n := range names {
			if got := runs[n].Load(); got != 1 {
				t.Errorf("migration %s ran %d times, want 1", n, got)
			}
		}
		assertTracking(t, db,
			trackRow{createUsers, 1}, trackRow{createPosts, 1}, trackRow{addBio, 1}, trackRow{insertAdmin, 1})
		if n := rowCount(t, db, "users"); n != 1 {
			t.Errorf("users rows = %d, want 1 (data migration ran more than once?)", n)
		}
	})
}

// TestLockTimeout holds the migration lock from one Migrator and checks that
// another one gives up with ErrLockTimeout without touching the database,
// then succeeds once the lock is released.
func TestLockTimeout(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		requireLockingBackend(t, b)
		peer := b.openPeer(t)

		holder := b.newMigrator(db)
		release, err := holder.Lock(context.Background())
		must(t, err, "holder Lock")
		released := false
		defer func() {
			if !released {
				_ = release()
			}
		}()

		waiter := b.newMigrator(peer, migrator.WithLockTimeout(500*time.Millisecond))
		register(t, waiter, createUsers)

		start := time.Now()
		err = waiter.Up()
		if !errors.Is(err, migrator.ErrLockTimeout) {
			t.Fatalf("Up while locked: err = %v, want ErrLockTimeout", err)
		}
		if d := time.Since(start); d < 500*time.Millisecond {
			t.Errorf("Up gave up after %v, before the 500ms lock timeout", d)
		}
		// Nothing ran: not even the tracking table was created.
		if tableExists(t, b, db, "migrations") || tableExists(t, b, db, "users") {
			t.Error("Up changed the database without holding the lock")
		}

		// Status does not take the lock and still works.
		st, err := waiter.Status()
		must(t, err, "Status while locked")
		if len(st) != 1 || st[0].Applied {
			t.Errorf("Status while locked = %+v, want one pending migration", st)
		}

		must(t, release(), "release")
		released = true

		must(t, waiter.Up(), "Up after release")
		assertTracking(t, db, trackRow{createUsers, 1})
	})
}

// TestLockRequiresTwoConnections checks the clear error for a pool limited to
// one connection, and that WithoutLock works with it.
func TestLockRequiresTwoConnections(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		requireLockingBackend(t, b)
		single := b.openPeer(t)
		single.SetMaxOpenConns(1)

		m := b.newMigrator(single)
		register(t, m, createUsers)
		err := m.Up()
		if err == nil {
			t.Fatal("Up with MaxOpenConns=1: expected an error")
		}
		if tableExists(t, b, db, "migrations") {
			t.Error("Up created the tracking table despite failing to lock")
		}

		unlocked := b.newMigrator(single, migrator.WithoutLock())
		register(t, unlocked, createUsers)
		must(t, unlocked.Up(), "Up WithoutLock")
		assertTracking(t, db, trackRow{createUsers, 1})
	})
}

// TestStatusIsReadOnly checks that Status on an empty database reports every
// migration as pending and creates nothing, not even the tracking table.
func TestStatusIsReadOnly(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		m := b.newMigrator(db)
		register(t, m, createUsers, createPosts)

		st, err := m.Status()
		must(t, err, "Status")
		if len(st) != 2 {
			t.Fatalf("Status returned %d entries, want 2", len(st))
		}
		for _, s := range st {
			if s.Applied || s.Batch != 0 || s.AppliedAt != nil {
				t.Errorf("Status reports %s applied on an empty database: %+v", s.Name, s)
			}
		}
		if tableExists(t, b, db, "migrations") {
			t.Error("Status created the tracking table")
		}

		// After Up it reads the table as before.
		must(t, m.Up(), "Up")
		st, err = m.Status()
		must(t, err, "Status after Up")
		for _, s := range st {
			if !s.Applied || s.Batch != 1 {
				t.Errorf("Status after Up: %+v", s)
			}
		}
	})
}
