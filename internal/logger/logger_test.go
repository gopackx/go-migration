package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- LogLevel tests ---

func TestLogLevelString(t *testing.T) {
	if "DEBUG" != LevelDebug.String() {
		t.Errorf("got %v, want %v", LevelDebug.String(), "DEBUG")
	}
	if "INFO" != LevelInfo.String() {
		t.Errorf("got %v, want %v", LevelInfo.String(), "INFO")
	}
	if "WARN" != LevelWarn.String() {
		t.Errorf("got %v, want %v", LevelWarn.String(), "WARN")
	}
	if "ERROR" != LevelError.String() {
		t.Errorf("got %v, want %v", LevelError.String(), "ERROR")
	}
	if "UNKNOWN" != LogLevel(99).String() {
		t.Errorf("got %v, want %v", LogLevel(99).String(), "UNKNOWN")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"  info  ", LevelInfo},
		{"unknown", LevelInfo}, // default
		{"", LevelInfo},        // default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if tt.expected != ParseLevel(tt.input) {
				t.Errorf("got %v, want %v", ParseLevel(tt.input), tt.expected)
			}
		})
	}
}

// --- ConsoleLogger tests ---

func TestConsoleLoggerOutputsAtCorrectLevels(t *testing.T) {
	var buf bytes.Buffer
	cl := newConsoleLoggerWithWriter(LevelInfo, &buf)

	cl.Info("hello %s", "world")
	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("%q not in %q", "[INFO]", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("%q not in %q", "hello world", output)
	}
}

func TestConsoleLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	cl := newConsoleLoggerWithWriter(LevelInfo, &buf)

	cl.Debug("should be suppressed")
	if len(buf.String()) != 0 {
		t.Errorf("expected empty, got %v", buf.String())
	}

	cl.Info("visible info")
	if !strings.Contains(buf.String(), "visible info") {
		t.Errorf("%q not in %q", "visible info", buf.String())
	}

	buf.Reset()
	cl.Warn("visible warn")
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("%q not in %q", "[WARN]", buf.String())
	}

	buf.Reset()
	cl.Error("visible error")
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("%q not in %q", "[ERROR]", buf.String())
	}
}

func TestConsoleLoggerDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	cl := newConsoleLoggerWithWriter(LevelDebug, &buf)

	cl.Debug("debug msg")
	if !strings.Contains(buf.String(), "[DEBUG]") {
		t.Errorf("%q not in %q", "[DEBUG]", buf.String())
	}
	if !strings.Contains(buf.String(), "debug msg") {
		t.Errorf("%q not in %q", "debug msg", buf.String())
	}
}

func TestConsoleLoggerErrorLevelSuppressesLower(t *testing.T) {
	var buf bytes.Buffer
	cl := newConsoleLoggerWithWriter(LevelError, &buf)

	cl.Debug("no")
	cl.Info("no")
	cl.Warn("no")
	if len(buf.String()) != 0 {
		t.Errorf("expected empty, got %v", buf.String())
	}

	cl.Error("yes")
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("%q not in %q", "[ERROR]", buf.String())
	}
}

func TestConsoleLoggerFormatIncludesTimestamp(t *testing.T) {
	var buf bytes.Buffer
	cl := newConsoleLoggerWithWriter(LevelDebug, &buf)

	cl.Info("test")
	// RFC3339 timestamps contain "T" and either "Z" or "+"/"-"
	line := buf.String()
	if !strings.Contains(line, "T") {
		t.Errorf("%q not in %q", "T", line)
	}
}

// --- FileLogger tests ---

func TestFileLoggerWritesToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	fl, err := NewFileLogger(path, LevelInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer fl.Close()

	fl.Info("file log entry")
	fl.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(data), "file log entry") {
		t.Errorf("%q not in %q", "file log entry", string(data))
	}
	if !strings.Contains(string(data), "[INFO]") {
		t.Errorf("%q not in %q", "[INFO]", string(data))
	}
}

func TestFileLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	fl := newFileLoggerWithWriter(LevelWarn, &buf)

	fl.Debug("no")
	fl.Info("no")
	if len(buf.String()) != 0 {
		t.Errorf("expected empty, got %v", buf.String())
	}

	fl.Warn("yes warn")
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("%q not in %q", "[WARN]", buf.String())
	}

	buf.Reset()
	fl.Error("yes error")
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("%q not in %q", "[ERROR]", buf.String())
	}
}

func TestFileLoggerInvalidPath(t *testing.T) {
	_, err := NewFileLogger("/nonexistent/dir/file.log", LevelInfo)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestFileLoggerCloseNilFile(t *testing.T) {
	fl := newFileLoggerWithWriter(LevelInfo, &bytes.Buffer{})
	if fl.Close() != nil {
		t.Errorf("unexpected error: %v", fl.Close())
	}
}

// --- NopLogger tests ---

func TestNopLoggerDoesNotPanic(t *testing.T) {
	var nop NopLogger
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		nop.Debug("test")
		nop.Info("test")
		nop.Warn("test")
		nop.Error("test")
	}()
}

// --- Interface compliance ---

func TestConsoleLoggerImplementsLogger(t *testing.T) {
	var _ Logger = (*ConsoleLogger)(nil)
}

func TestFileLoggerImplementsLogger(t *testing.T) {
	var _ Logger = (*FileLogger)(nil)
}

func TestNopLoggerImplementsLogger(t *testing.T) {
	var _ Logger = NopLogger{}
}

// --- formatEntry ---

func TestFormatEntryContainsAllParts(t *testing.T) {
	entry := formatEntry(LevelInfo, "migration %s %s", "users", "up")
	if !strings.Contains(entry, "[INFO]") {
		t.Errorf("%q not in %q", "[INFO]", entry)
	}
	if !strings.Contains(entry, "migration users up") {
		t.Errorf("%q not in %q", "migration users up", entry)
	}
	if !strings.HasSuffix(entry, "\n") {
		t.Errorf("expected true")
	}
}
