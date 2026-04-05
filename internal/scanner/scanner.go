// Package scanner provides functions to discover migration and seeder
// files in a directory by matching known naming patterns.
package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// migrationPattern matches files named YYYYMMDDHHMMSS_<description>.go
var migrationPattern = regexp.MustCompile(`^\d{14}_.+\.go$`)

// ScanMigrations scans dir for Go files whose names match the migration
// pattern YYYYMMDDHHMMSS_*.go. It returns a sorted list of full file paths.
// Test files (*_test.go) are excluded.
func ScanMigrations(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if migrationPattern.MatchString(name) {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

// ScanSeeders scans dir for Go files whose names match the seeder
// pattern *_seeder.go. It returns a sorted list of full file paths.
// Test files (*_test.go) are excluded.
func ScanSeeders(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if strings.HasSuffix(name, "_seeder.go") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files, nil
}
