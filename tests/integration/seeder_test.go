//go:build integration

package integration_test

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/pkg/seeder"
)

// Seeder names are chosen so that alphabetical order (which the runner uses
// to iterate) is the reverse of dependency order.
const (
	seedSummary = "a_summary"
	seedPosts   = "b_posts"
	seedUsers   = "c_users"
)

type testSeeder struct {
	deps []string
	run  func(db *sql.DB) error
}

func (s testSeeder) Run(db *sql.DB) error { return s.run(db) }
func (s testSeeder) DependsOn() []string  { return s.deps }

var _ seeder.DependentSeeder = testSeeder{}

const seededUsers = 5

// seedRegistry builds a registry of three chained seeders that record the
// order in which they run into *order.
func seedRegistry(t *testing.T, b backend, order *[]string) *seeder.Registry {
	t.Helper()
	reg := seeder.NewRegistry()

	must(t, reg.Register(seedSummary, testSeeder{
		deps: []string{seedPosts, seedUsers},
		run: func(db *sql.DB) error {
			*order = append(*order, seedSummary)
			var users, posts int
			if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); err != nil {
				return err
			}
			if err := db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&posts); err != nil {
				return err
			}
			if users != seededUsers || posts != seededUsers {
				return fmt.Errorf("summary ran before its dependencies: users=%d posts=%d", users, posts)
			}
			return nil
		},
	}), "register summary")

	must(t, reg.Register(seedPosts, testSeeder{
		deps: []string{seedUsers},
		run: func(db *sql.DB) error {
			*order = append(*order, seedPosts)
			rows, err := db.Query("SELECT id, email FROM users ORDER BY email")
			if err != nil {
				return err
			}
			defer rows.Close()
			var records []map[string]any
			for rows.Next() {
				var id int64
				var email string
				if err := rows.Scan(&id, &email); err != nil {
					return err
				}
				records = append(records, map[string]any{"user_id": id, "title": "post by " + email})
			}
			if err := rows.Err(); err != nil {
				return err
			}
			if len(records) == 0 {
				return errors.New("posts seeder ran before users seeder")
			}
			return seeder.CreateManyWithDialect(db, "posts", records, 0, b.dialect)
		},
	}), "register posts")

	must(t, reg.Register(seedUsers, testSeeder{
		run: func(db *sql.DB) error {
			*order = append(*order, seedUsers)
			records := make([]map[string]any, seededUsers)
			for i := range records {
				records[i] = map[string]any{
					"name":  fmt.Sprintf("User %d", i+1),
					"email": fmt.Sprintf("user%d@example.com", i+1),
				}
			}
			// chunkSize 2 forces three INSERT statements (2+2+1).
			return seeder.CreateManyWithDialect(db, "users", records, 2, b.dialect)
		},
	}), "register users")

	return reg
}

func migrateUsersAndPosts(t *testing.T, b backend, db *sql.DB) {
	t.Helper()
	m := b.newMigrator(db)
	register(t, m, createUsers, createPosts)
	must(t, m.Up(), "Up")
}

// TestSeederRunAll checks dependency ordering and the data written by
// CreateManyWithDialect.
func TestSeederRunAll(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		migrateUsersAndPosts(t, b, db)

		var order []string
		runner := seeder.NewRunner(seedRegistry(t, b, &order), db, nil)
		must(t, runner.RunAll(), "RunAll")

		want := []string{seedUsers, seedPosts, seedSummary}
		if strings.Join(order, ",") != strings.Join(want, ",") {
			t.Errorf("run order = %v, want %v", order, want)
		}
		if n := rowCount(t, db, "users"); n != seededUsers {
			t.Errorf("users rows = %d, want %d", n, seededUsers)
		}
		if n := rowCount(t, db, "posts"); n != seededUsers {
			t.Errorf("posts rows = %d, want %d", n, seededUsers)
		}

		var name string
		must(t, db.QueryRow(b.q("SELECT name FROM users WHERE email = ?"), "user3@example.com").Scan(&name), "select user3")
		if name != "User 3" {
			t.Errorf("user3 name = %q, want %q", name, "User 3")
		}

		// Every post belongs to the user named in its title.
		var mismatched int
		must(t, db.QueryRow(
			"SELECT COUNT(*) FROM posts p JOIN users u ON u.id = p.user_id WHERE p.title <> "+concat(b, "'post by '", "u.email"),
		).Scan(&mismatched), "join posts/users")
		if mismatched != 0 {
			t.Errorf("%d posts reference the wrong user", mismatched)
		}
	})
}

// TestSeederTruncate checks that Truncate quotes the table name for the
// backend's dialect (backticks on MySQL, double quotes elsewhere).
func TestSeederTruncate(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		migrateUsersAndPosts(t, b, db)

		var order []string
		runner := seeder.NewRunner(seedRegistry(t, b, &order), db, nil, seeder.WithDialect(b.dialect))
		must(t, runner.RunAll(), "RunAll")

		// posts first: it references users.
		must(t, runner.Truncate("posts"), "Truncate posts")
		if n := rowCount(t, db, "posts"); n != 0 {
			t.Errorf("posts rows after Truncate = %d, want 0", n)
		}
		if n := rowCount(t, db, "users"); n != seededUsers {
			t.Errorf("users rows after truncating posts = %d, want %d", n, seededUsers)
		}
		must(t, runner.Truncate("users"), "Truncate users")
		if n := rowCount(t, db, "users"); n != 0 {
			t.Errorf("users rows after Truncate = %d, want 0", n)
		}

		if err := runner.Truncate("no_such_table"); err == nil {
			t.Error("Truncate(no_such_table) succeeded")
		}
	})
}

// TestSeederRunWithDependencies checks that Run(name) runs exactly the
// seeder's transitive dependencies first, and nothing else.
func TestSeederRunWithDependencies(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		migrateUsersAndPosts(t, b, db)

		var order []string
		runner := seeder.NewRunner(seedRegistry(t, b, &order), db, nil)
		must(t, runner.Run(seedPosts), "Run posts")

		want := []string{seedUsers, seedPosts}
		if strings.Join(order, ",") != strings.Join(want, ",") {
			t.Errorf("run order = %v, want %v", order, want)
		}
		if n := rowCount(t, db, "posts"); n != seededUsers {
			t.Errorf("posts rows = %d, want %d", n, seededUsers)
		}

		if err := runner.Run("missing"); !errors.Is(err, seeder.ErrSeederNotFound) {
			t.Errorf("Run(missing) err = %v, want ErrSeederNotFound", err)
		}
	})
}

// TestSeederCircularDependency checks that a cycle is reported before any
// seeder runs.
func TestSeederCircularDependency(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend, db *sql.DB) {
		ran := false
		run := func(*sql.DB) error { ran = true; return nil }
		reg := seeder.NewRegistry()
		must(t, reg.Register("x", testSeeder{deps: []string{"y"}, run: run}), "register x")
		must(t, reg.Register("y", testSeeder{deps: []string{"x"}, run: run}), "register y")

		err := seeder.NewRunner(reg, db, nil).RunAll()
		if !errors.Is(err, seeder.ErrCircularDependency) {
			t.Fatalf("RunAll err = %v, want ErrCircularDependency", err)
		}
		if ran {
			t.Error("a seeder ran despite the dependency cycle")
		}
	})
}

// concat returns a SQL string concatenation expression for the backend.
func concat(b backend, a, c string) string {
	if b.name == "mysql" {
		return "CONCAT(" + a + ", " + c + ")"
	}
	return a + " || " + c
}
