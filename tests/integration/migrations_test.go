//go:build integration

package integration_test

import (
	"errors"
	"fmt"

	"github.com/gopackx/go-migration/pkg/migrator"
	"github.com/gopackx/go-migration/pkg/schema"
)

// Migration names used across the scenarios.
const (
	createUsers  = "20240101000001_create_users_table"
	createPosts  = "20240101000002_create_posts_table"
	addBio       = "20240101000003_add_bio_to_users_table"
	failingDML   = "20240101000004_failing_insert"
	failingDDL   = "20240101000005_failing_create"
	nonTxIndex   = "20240101000006_non_transactional_index"
	nonTxFailure = "20240101000007_non_transactional_failure"
	alterUsers   = "20240101000008_alter_users_columns"
	txIndex      = "20240101000009_transactional_concurrent_index"
)

var errBoom = errors.New("boom")

// funcMigration adapts two closures to migrator.Migration.
type funcMigration struct {
	up   func(s *schema.Builder) error
	down func(s *schema.Builder) error
}

func (m funcMigration) Up(s *schema.Builder) error   { return m.up(s) }
func (m funcMigration) Down(s *schema.Builder) error { return m.down(s) }

// nonTxMigration is a funcMigration that opts out of the transaction.
type nonTxMigration struct{ funcMigration }

func (nonTxMigration) DisableTransaction() bool { return true }

var _ migrator.TransactionOption = nonTxMigration{}

func createUsersMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			return s.Create("users", func(t *schema.Blueprint) {
				t.ID()
				t.String("name", 100)
				t.String("email", 191)
				t.UniqueIndex("email")
				t.Index("name")
			})
		},
		down: func(s *schema.Builder) error { return s.DropIfExists("users") },
	}
}

func createPostsMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			return s.Create("posts", func(t *schema.Blueprint) {
				t.ID()
				t.BigInteger("user_id").Unsigned()
				t.String("title", 200)
				t.Foreign("user_id").References("id").On("users").OnDeleteAction("CASCADE")
			})
		},
		down: func(s *schema.Builder) error { return s.DropIfExists("posts") },
	}
}

// addBioMigration adds users.bio and checks HasColumn before and after, on
// the migration's own executor.
func addBioMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			has, err := s.HasColumn("users", "bio")
			if err != nil {
				return err
			}
			if has {
				return fmt.Errorf("users.bio exists before the migration")
			}
			if err := s.Alter("users", func(t *schema.Blueprint) {
				t.Text("bio").Nullable()
			}); err != nil {
				return err
			}
			has, err = s.HasColumn("users", "bio")
			if err != nil {
				return err
			}
			if !has {
				return fmt.Errorf("HasColumn(users, bio) = false after ALTER")
			}
			return nil
		},
		down: func(s *schema.Builder) error {
			return s.Alter("users", func(t *schema.Blueprint) {
				t.DropColumn("bio")
			})
		},
	}
}

// alterUsersMigration adds, renames and (on Down) drops a column, and uses
// HasTable/HasColumn from inside the migration.
func alterUsersMigration() migrator.Migration {
	return funcMigration{
		up: func(s *schema.Builder) error {
			ok, err := s.HasTable("users")
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("HasTable(users) = false")
			}
			ok, err = s.HasTable("no_such_table")
			if err != nil {
				return err
			}
			if ok {
				return fmt.Errorf("HasTable(no_such_table) = true")
			}

			if err := s.Alter("users", func(t *schema.Blueprint) {
				t.String("nickname", 50).Nullable()
			}); err != nil {
				return fmt.Errorf("add nickname: %w", err)
			}
			if ok, err := s.HasColumn("users", "nickname"); err != nil || !ok {
				return fmt.Errorf("HasColumn(users, nickname) = %v, %v", ok, err)
			}
			if err := s.Alter("users", func(t *schema.Blueprint) {
				t.RenameColumn("nickname", "handle")
			}); err != nil {
				return fmt.Errorf("rename nickname: %w", err)
			}
			if ok, err := s.HasColumn("users", "nickname"); err != nil || ok {
				return fmt.Errorf("HasColumn(users, nickname) after rename = %v, %v", ok, err)
			}
			if ok, err := s.HasColumn("users", "handle"); err != nil || !ok {
				return fmt.Errorf("HasColumn(users, handle) = %v, %v", ok, err)
			}
			return nil
		},
		down: func(s *schema.Builder) error {
			return s.Alter("users", func(t *schema.Blueprint) {
				t.DropColumn("handle")
			})
		},
	}
}

func noopDown(*schema.Builder) error { return nil }
