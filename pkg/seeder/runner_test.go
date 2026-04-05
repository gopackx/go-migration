package seeder

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"strings"

	"github.com/DATA-DOG/go-sqlmock"
)

// trackingSeeder records when it was run for order verification.
type trackingSeeder struct {
	name  string
	order *[]string
}

func (s *trackingSeeder) Run(db *sql.DB) error {
	*s.order = append(*s.order, s.name)
	return nil
}

// dependentTrackingSeeder is a seeder with dependencies that tracks execution.
type dependentTrackingSeeder struct {
	name  string
	deps  []string
	order *[]string
}

func (s *dependentTrackingSeeder) Run(db *sql.DB) error {
	*s.order = append(*s.order, s.name)
	return nil
}

func (s *dependentTrackingSeeder) DependsOn() []string {
	return s.deps
}

// failingSeeder always returns an error.
type failingSeeder struct {
	name string
	err  error
}

func (s *failingSeeder) Run(db *sql.DB) error {
	return s.err
}

// testLogger captures log messages for verification.
type testLogger struct {
	infos  []string
	errors []string
}

func (l *testLogger) Info(msg string, args ...any) {
	l.infos = append(l.infos, fmt.Sprintf(msg, args...))
}

func (l *testLogger) Error(msg string, args ...any) {
	l.errors = append(l.errors, fmt.Sprintf(msg, args...))
}

func newTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestNewRunner(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	logger := &testLogger{}

	runner := NewRunner(reg, db, logger)
	if runner == nil {
		t.Errorf("expected non-nil")
	}
}

func TestRunAllNoDependencies(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}
	if reg.Register("posts", &trackingSeeder{name: "posts", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("posts", &trackingSeeder{name: "posts", order: &order}))
	}
	if reg.Register("comments", &trackingSeeder{name: "comments", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("comments", &trackingSeeder{name: "comments", order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	err := runner.RunAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All seeders should have been executed
	if len(order) != 3 {
		t.Errorf("len = %d, want %d", len(order), 3)
	}
	// helper to check if a string slice contains a value
	containsStr := func(slice []string, s string) bool {
		for _, v := range slice {
			if v == s {
				return true
			}
		}
		return false
	}
	if !containsStr(order, "users") {
		t.Errorf("%q not in %v", "users", order)
	}
	if !containsStr(order, "posts") {
		t.Errorf("%q not in %v", "posts", order)
	}
	if !containsStr(order, "comments") {
		t.Errorf("%q not in %v", "comments", order)
	}
}

func TestRunAllWithDependencies(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	// comments depends on posts, posts depends on users
	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}
	if reg.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}))
	}
	if reg.Register("comments", &dependentTrackingSeeder{name: "comments", deps: []string{"posts"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("comments", &dependentTrackingSeeder{name: "comments", deps: []string{"posts"}, order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	err := runner.RunAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("len = %d, want %d", len(order), 3)
	}
	// users must come before posts, posts must come before comments
	usersIdx := indexOf(order, "users")
	postsIdx := indexOf(order, "posts")
	commentsIdx := indexOf(order, "comments")
	if usersIdx >= postsIdx {
		t.Errorf("users should run before posts")
	}
	if postsIdx >= commentsIdx {
		t.Errorf("posts should run before comments")
	}
}

func TestRunSingleWithDependencies(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}
	if reg.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}))
	}
	if reg.Register("comments", &dependentTrackingSeeder{name: "comments", deps: []string{"posts"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("comments", &dependentTrackingSeeder{name: "comments", deps: []string{"posts"}, order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	// Running "comments" should also run users and posts first
	err := runner.Run("comments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedOrder := []string{"users", "posts", "comments"}
	if len(order) != len(expectedOrder) {
		t.Errorf("len = %d, want %d", len(order), len(expectedOrder))
	}
	for i := range expectedOrder {
		if i < len(order) && order[i] != expectedOrder[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], expectedOrder[i])
		}
	}
}

func TestRunSingleNoDependencies(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}
	if reg.Register("posts", &trackingSeeder{name: "posts", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("posts", &trackingSeeder{name: "posts", order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	err := runner.Run("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only "users" should have run
	if !reflect.DeepEqual([]string{"users"}, order) {
		t.Errorf("got %v, want %v", order, []string{"users"})
	}
}

func TestCircularDependencyDetection(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	// A -> B -> C -> A (cycle)
	if reg.Register("a", &dependentTrackingSeeder{name: "a", deps: []string{"c"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("a", &dependentTrackingSeeder{name: "a", deps: []string{"c"}, order: &order}))
	}
	if reg.Register("b", &dependentTrackingSeeder{name: "b", deps: []string{"a"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("b", &dependentTrackingSeeder{name: "b", deps: []string{"a"}, order: &order}))
	}
	if reg.Register("c", &dependentTrackingSeeder{name: "c", deps: []string{"b"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("c", &dependentTrackingSeeder{name: "c", deps: []string{"b"}, order: &order}))
	}

	runner := NewRunner(reg, db, nil)

	// RunAll should detect the cycle
	err := runner.RunAll()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("expected true")
	}

	// Run single should also detect the cycle
	order = nil
	err = runner.Run("a")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("expected true")
	}
}

func TestSeederFailureStopsExecution(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string
	seederErr := fmt.Errorf("seed failed")

	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}
	if reg.Register("posts", &failingSeeder{name: "posts", err: seederErr}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("posts", &failingSeeder{name: "posts", err: seederErr}))
	}
	if reg.Register("tags", &trackingSeeder{name: "tags", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("tags", &trackingSeeder{name: "tags", order: &order}))
	}

	// posts depends on users, tags depends on posts
	// Replace with dependent seeders to ensure order
	reg2 := NewRegistry()
	if reg2.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg2.Register("users", &trackingSeeder{name: "users", order: &order}))
	}
	if reg2.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg2.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}))
	}

	// Use a simpler scenario: users succeeds, then failing_seeder fails
	reg3 := NewRegistry()
	var order3 []string
	if reg3.Register("aaa_first", &trackingSeeder{name: "aaa_first", order: &order3}) != nil {
		t.Fatalf("unexpected error: %v", reg3.Register("aaa_first", &trackingSeeder{name: "aaa_first", order: &order3}))
	}
	if reg3.Register("bbb_failing", &failingSeeder{name: "bbb_failing", err: seederErr}) != nil {
		t.Fatalf("unexpected error: %v", reg3.Register("bbb_failing", &failingSeeder{name: "bbb_failing", err: seederErr}))
	}
	if reg3.Register("ccc_last", &trackingSeeder{name: "ccc_last", order: &order3}) != nil {
		t.Fatalf("unexpected error: %v", reg3.Register("ccc_last", &trackingSeeder{name: "ccc_last", order: &order3}))
	}

	runner := NewRunner(reg3, db, nil)
	err := runner.RunAll()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bbb_failing") {
		t.Errorf("%q not in %q", "bbb_failing", err.Error())
	}
	if !errors.Is(err, seederErr) {
		t.Errorf("got %v, want %v", err, seederErr)
	}

	// "aaa_first" should have run, "ccc_last" should NOT have run
	containsStr3 := func(slice []string, s string) bool {
		for _, v := range slice {
			if v == s {
				return true
			}
		}
		return false
	}
	if !containsStr3(order3, "aaa_first") {
		t.Errorf("%q not in %v", "aaa_first", order3)
	}
	if containsStr3(order3, "ccc_last") {
		t.Errorf("%q should not be in %v", "ccc_last", order3)
	}
}

func TestRunNonExistentSeeder(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)

	runner := NewRunner(reg, db, nil)
	err := runner.Run("nonexistent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrSeederNotFound) {
		t.Errorf("expected true")
	}
}

func TestRunAllEmptyRegistry(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)

	runner := NewRunner(reg, db, nil)
	err := runner.RunAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWithLogger(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	logger := &testLogger{}
	var order []string

	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}

	runner := NewRunner(reg, db, logger)
	err := runner.RunAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.infos) == 0 {
		t.Errorf("expected non-empty log entries")
	}
}

func TestRunWithNilLogger(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	if reg.Register("users", &trackingSeeder{name: "users", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &trackingSeeder{name: "users", order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	// Should not panic with nil logger
	err := runner.RunAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCircularDependencyErrorMessage(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	// Self-referencing seeder
	if reg.Register("self", &dependentTrackingSeeder{name: "self", deps: []string{"self"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("self", &dependentTrackingSeeder{name: "self", deps: []string{"self"}, order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	err := runner.Run("self")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("expected true")
	}
	if !strings.Contains(err.Error(), "self") {
		t.Errorf("%q not in %q", "self", err.Error())
	}
}

func TestDiamondDependency(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string

	// Diamond: D depends on B and C, both B and C depend on A
	if reg.Register("a", &trackingSeeder{name: "a", order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("a", &trackingSeeder{name: "a", order: &order}))
	}
	if reg.Register("b", &dependentTrackingSeeder{name: "b", deps: []string{"a"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("b", &dependentTrackingSeeder{name: "b", deps: []string{"a"}, order: &order}))
	}
	if reg.Register("c", &dependentTrackingSeeder{name: "c", deps: []string{"a"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("c", &dependentTrackingSeeder{name: "c", deps: []string{"a"}, order: &order}))
	}
	if reg.Register("d", &dependentTrackingSeeder{name: "d", deps: []string{"b", "c"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("d", &dependentTrackingSeeder{name: "d", deps: []string{"b", "c"}, order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	err := runner.Run("d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A should run first, then B and C (in some order), then D last
	if len(order) != 4 {
		t.Errorf("len = %d, want %d", len(order), 4)
	}
	if "a" != order[0] {
		t.Errorf("a should run first")
	}
	if "d" != order[3] {
		t.Errorf("d should run last")
	}
	// Each seeder should run exactly once
	if len(order) != len([]string{"a", "b", "c", "d"}) {
		t.Errorf("len = %d, want %d", len(order), len([]string{"a", "b", "c", "d"}))
	}
}

func TestSeederFailureInRunStopsExecution(t *testing.T) {
	reg := NewRegistry()
	db, _ := newTestDB(t)
	var order []string
	seederErr := fmt.Errorf("users seed failed")

	// posts depends on users, users fails
	if reg.Register("users", &failingSeeder{name: "users", err: seederErr}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("users", &failingSeeder{name: "users", err: seederErr}))
	}
	if reg.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}) != nil {
		t.Fatalf("unexpected error: %v", reg.Register("posts", &dependentTrackingSeeder{name: "posts", deps: []string{"users"}, order: &order}))
	}

	runner := NewRunner(reg, db, nil)
	err := runner.Run("posts")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, seederErr) {
		t.Errorf("got %v, want %v", err, seederErr)
	}
	if !strings.Contains(err.Error(), "users") {
		t.Errorf("%q not in %q", "users", err.Error())
	}
	// posts should not have run
	if len(order) != 0 {
		t.Errorf("expected empty, got %v", order)
	}
}

// indexOf returns the index of s in slice, or -1 if not found.
func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}
