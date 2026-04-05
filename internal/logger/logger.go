package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	// LevelDebug is the most verbose log level.
	LevelDebug LogLevel = iota
	// LevelInfo is the default log level for informational messages.
	LevelInfo
	// LevelWarn indicates a potential issue.
	LevelWarn
	// LevelError indicates a failure.
	LevelError
)

// String returns the human-readable name of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a LogLevel. It is case-insensitive.
// Unrecognised strings default to LevelInfo.
func ParseLevel(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger defines the logging interface used throughout go-migration.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// ConsoleLogger writes log entries to os.Stdout with level filtering.
type ConsoleLogger struct {
	level LogLevel
	out   io.Writer
	mu    sync.Mutex
}

// NewConsoleLogger creates a ConsoleLogger that only emits messages at or above
// the given level. Output goes to os.Stdout.
func NewConsoleLogger(level LogLevel) *ConsoleLogger {
	return &ConsoleLogger{level: level, out: os.Stdout}
}

// newConsoleLoggerWithWriter is an internal constructor used for testing.
func newConsoleLoggerWithWriter(level LogLevel, w io.Writer) *ConsoleLogger {
	return &ConsoleLogger{level: level, out: w}
}

func (c *ConsoleLogger) Debug(msg string, args ...any) { c.log(LevelDebug, msg, args...) }
func (c *ConsoleLogger) Info(msg string, args ...any)  { c.log(LevelInfo, msg, args...) }
func (c *ConsoleLogger) Warn(msg string, args ...any)  { c.log(LevelWarn, msg, args...) }
func (c *ConsoleLogger) Error(msg string, args ...any) { c.log(LevelError, msg, args...) }

func (c *ConsoleLogger) log(lvl LogLevel, msg string, args ...any) {
	if lvl < c.level {
		return
	}
	entry := formatEntry(lvl, msg, args...)
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprint(c.out, entry)
}

// FileLogger writes log entries to a file with level filtering.
type FileLogger struct {
	level LogLevel
	out   io.Writer
	file  *os.File
	mu    sync.Mutex
}

// NewFileLogger creates a FileLogger that appends to the file at path.
// Only messages at or above the given level are written.
func NewFileLogger(path string, level LogLevel) (*FileLogger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}
	return &FileLogger{level: level, out: f, file: f}, nil
}

// newFileLoggerWithWriter is an internal constructor used for testing.
func newFileLoggerWithWriter(level LogLevel, w io.Writer) *FileLogger {
	return &FileLogger{level: level, out: w}
}

func (fl *FileLogger) Debug(msg string, args ...any) { fl.log(LevelDebug, msg, args...) }
func (fl *FileLogger) Info(msg string, args ...any)  { fl.log(LevelInfo, msg, args...) }
func (fl *FileLogger) Warn(msg string, args ...any)  { fl.log(LevelWarn, msg, args...) }
func (fl *FileLogger) Error(msg string, args ...any) { fl.log(LevelError, msg, args...) }

// Close closes the underlying file. It is a no-op if the FileLogger was
// created with a plain io.Writer (testing path).
func (fl *FileLogger) Close() error {
	if fl.file != nil {
		return fl.file.Close()
	}
	return nil
}

func (fl *FileLogger) log(lvl LogLevel, msg string, args ...any) {
	if lvl < fl.level {
		return
	}
	entry := formatEntry(lvl, msg, args...)
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fmt.Fprint(fl.out, entry)
}

// NopLogger is a logger that silently discards all messages.
type NopLogger struct{}

func (NopLogger) Debug(string, ...any) {}
func (NopLogger) Info(string, ...any)  {}
func (NopLogger) Warn(string, ...any)  {}
func (NopLogger) Error(string, ...any) {}

// formatEntry builds a single log line: "2006-01-02T15:04:05Z07:00 [LEVEL] message\n"
func formatEntry(lvl LogLevel, msg string, args ...any) string {
	ts := time.Now().Format(time.RFC3339)
	text := fmt.Sprintf(msg, args...)
	return fmt.Sprintf("%s [%s] %s\n", ts, lvl, text)
}
