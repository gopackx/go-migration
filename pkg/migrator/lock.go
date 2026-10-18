package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/gopackx/go-migration/pkg/schema"
)

// DefaultLockTimeout is how long Up, Rollback, Reset, Refresh and Fresh wait
// for the migration lock before failing with ErrLockTimeout.
const DefaultLockTimeout = 30 * time.Second

// Polling interval bounds while waiting for the migration lock.
const (
	lockInitialBackoff = 100 * time.Millisecond
	lockMaxBackoff     = time.Second
)

// WithLockTimeout sets how long the Migrator waits for the cross-process
// migration lock (default DefaultLockTimeout). A zero or negative duration
// means a single attempt: fail immediately if another process holds the lock.
func WithLockTimeout(d time.Duration) Option {
	return func(m *Migrator) {
		m.lockTimeout = d
	}
}

// WithoutLock disables the cross-process migration lock. Use it only when a
// single process can ever run migrations against the database, or when the
// connection pool cannot provide the extra connection the lock needs.
func WithoutLock() Option {
	return func(m *Migrator) {
		m.lockDisabled = true
	}
}

// lockKey is the name of the migration lock. It includes the tracking table
// name so that independent migration sets in one database do not block each
// other.
func (m *Migrator) lockKey() string {
	return "go-migration:" + m.tableName
}

// lockGrammar returns the grammar used for the lock queries: the configured
// grammar, or the Tracker's PostgreSQL fallback when none was configured.
func (m *Migrator) lockGrammar() schema.Grammar {
	if m.grammar != nil {
		return m.grammar
	}
	return m.tracker.grammar
}

// Lock acquires the cross-process migration lock and returns a function that
// releases it. Up, Rollback, Reset, Refresh and Fresh take this lock
// themselves; Lock is exported so that other writers (such as seeders) can be
// serialised with migrations.
//
// The lock is a session-level database lock (PostgreSQL advisory lock, MySQL
// GET_LOCK) held on one connection pinned from the pool for as long as the
// lock is held, so the pool must allow at least two open connections: one for
// the lock and one for the work done under it. A pool limited to one
// connection is reported as an error rather than deadlocking.
//
// The lock is polled with a short backoff until it is acquired or the lock
// timeout elapses, in which case the error wraps ErrLockTimeout.
//
// If the grammar has no lock query (SQLite, whose file lock already
// serialises writers) or locking was disabled with WithoutLock, no connection
// is pinned and a no-op release function is returned.
//
// The release function runs the unlock query on the pinned connection and
// returns the connection to the pool. If unlocking fails, the connection is
// discarded instead, which makes the server drop the lock when the physical
// connection closes; the error is still returned. Calling release more than
// once is safe.
func (m *Migrator) Lock(ctx context.Context) (release func() error, err error) {
	noop := func() error { return nil }
	if m.lockDisabled || m.db == nil {
		return noop, nil
	}

	g := m.lockGrammar()
	key := m.lockKey()
	lockQuery := g.CompileTryLock(key)
	if lockQuery == "" {
		return noop, nil
	}
	unlockQuery := g.CompileUnlock(key)

	if m.db.Stats().MaxOpenConnections == 1 {
		return nil, fmt.Errorf("migration lock: the connection pool is limited to 1 open connection, " +
			"but the lock needs its own connection in addition to the one running migrations; " +
			"set MaxOpenConns to at least 2 or disable locking (WithoutLock / --no-lock)")
	}

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration lock: acquire connection: %w", err)
	}

	timeout := m.lockTimeout
	deadline := time.Now().Add(timeout)
	backoff := lockInitialBackoff
	for {
		acquired, err := queryLockResult(ctx, conn, lockQuery, key)
		if err != nil {
			// The outcome is unknown; make sure a lock taken by this session
			// cannot outlive it by closing the physical connection.
			discardConn(conn)
			_ = conn.Close()
			return nil, fmt.Errorf("migration lock: acquire %q: %w", key, err)
		}
		if acquired {
			break
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			_ = conn.Close()
			return nil, fmt.Errorf("%w within %s (lock %q is held by another process)", ErrLockTimeout, timeout, key)
		}
		wait := backoff
		if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = conn.Close()
			return nil, fmt.Errorf("migration lock: %w", ctx.Err())
		case <-timer.C:
		}
		backoff *= 2
		if backoff > lockMaxBackoff {
			backoff = lockMaxBackoff
		}
	}

	var once sync.Once
	var releaseErr error
	release = func() error {
		once.Do(func() {
			releaseErr = unlock(conn, unlockQuery, key)
		})
		return releaseErr
	}
	return release, nil
}

// unlock releases the lock on conn and closes conn. On any failure conn is
// discarded so that the server releases the lock with the session.
func unlock(conn *sql.Conn, query, key string) error {
	released, err := queryLockResult(context.Background(), conn, query, key)
	if err == nil && !released {
		err = fmt.Errorf("lock was not held by this session")
	}
	if err != nil {
		discardConn(conn)
		_ = conn.Close()
		return fmt.Errorf("migration lock: release %q (connection discarded): %w", key, err)
	}
	return conn.Close()
}

// queryLockResult runs a lock/unlock query with key as its only argument and
// interprets the single boolean-ish result: PostgreSQL returns a boolean,
// MySQL 1/0 or NULL (GET_LOCK on error). sql.NullBool accepts all of these;
// NULL and false/0 mean "not acquired / not released".
func queryLockResult(ctx context.Context, conn *sql.Conn, query, key string) (bool, error) {
	var res sql.NullBool
	if err := conn.QueryRowContext(ctx, query, key).Scan(&res); err != nil {
		return false, err
	}
	return res.Valid && res.Bool, nil
}

// withLock runs fn while holding the migration lock. A failure to release the
// lock does not fail an otherwise successful operation: the lock connection
// has then been discarded, so the server drops the lock when it closes. The
// failure is logged instead.
func (m *Migrator) withLock(fn func() error) error {
	release, err := m.Lock(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		if rerr := release(); rerr != nil && m.logger != nil {
			m.logger.Error("%v", rerr)
		}
	}()
	return fn()
}
