package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// helper to create empty files in a temp directory.
func touch(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanMigrations(t *testing.T) {
	dir := t.TempDir()

	touch(t, dir, "20240101120000_create_users.go")
	touch(t, dir, "20240201120000_add_email.go")
	touch(t, dir, "helper.go")
	touch(t, dir, "20240101120000_create_users_test.go")
	if err := os.Mkdir(filepath.Join(dir, "20240301120000_subdir.go"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files, err := ScanMigrations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		filepath.Join(dir, "20240101120000_create_users.go"),
		filepath.Join(dir, "20240201120000_add_email.go"),
	}
	if !reflect.DeepEqual(expected, files) {
		t.Errorf("got %v, want %v", files, expected)
	}
}

func TestScanMigrations_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := ScanMigrations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestScanMigrations_NonExistentDir(t *testing.T) {
	_, err := ScanMigrations("/nonexistent_dir_12345")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestScanSeeders(t *testing.T) {
	dir := t.TempDir()

	touch(t, dir, "users_seeder.go")
	touch(t, dir, "posts_seeder.go")
	touch(t, dir, "helper.go")
	touch(t, dir, "users_seeder_test.go")

	files, err := ScanSeeders(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		filepath.Join(dir, "posts_seeder.go"),
		filepath.Join(dir, "users_seeder.go"),
	}
	if !reflect.DeepEqual(expected, files) {
		t.Errorf("got %v, want %v", files, expected)
	}
}

func TestScanSeeders_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := ScanSeeders(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestScanSeeders_NonExistentDir(t *testing.T) {
	_, err := ScanSeeders("/nonexistent_dir_12345")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestScanMigrations_SortOrder(t *testing.T) {
	dir := t.TempDir()

	touch(t, dir, "20240301120000_third.go")
	touch(t, dir, "20240101120000_first.go")
	touch(t, dir, "20240201120000_second.go")

	files, err := ScanMigrations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		filepath.Join(dir, "20240101120000_first.go"),
		filepath.Join(dir, "20240201120000_second.go"),
		filepath.Join(dir, "20240301120000_third.go"),
	}
	if !reflect.DeepEqual(expected, files) {
		t.Errorf("got %v, want %v", files, expected)
	}
}
