package cli

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/gopackx/go-migration/internal/logger"
	"github.com/gopackx/go-migration/pkg/config"
)

func TestNewApp(t *testing.T) {
	app := NewApp(nil)

	if app == nil {
		t.Fatal("expected non-nil app")
	}
	if app.Router() == nil {
		t.Fatal("expected non-nil router")
	}
	if app.Context() != nil {
		t.Errorf("expected nil context, got %v", app.Context())
	}
}

func TestNewAppWithContext(t *testing.T) {
	ctx := &CLIContext{
		Config: &config.Config{DefaultConn: "test"},
		Logger: logger.NopLogger{},
	}
	app := NewApp(ctx)

	if app.Context() != ctx {
		t.Errorf("got %v, want %v", app.Context(), ctx)
	}
}

func TestApp_SetContext(t *testing.T) {
	app := NewApp(nil)
	if app.Context() != nil {
		t.Errorf("expected nil context initially")
	}

	ctx := &CLIContext{
		Config: &config.Config{DefaultConn: "updated"},
	}
	app.SetContext(ctx)

	if app.Context() != ctx {
		t.Errorf("got %v, want %v", app.Context(), ctx)
	}
}

func TestApp_RunHelp(t *testing.T) {
	out := &bytes.Buffer{}
	router := NewRouter(out, out, nil)
	app := &App{router: router}

	router.Register("test", "A test command", func(args []string, flags *flag.FlagSet) error {
		return nil
	}, nil)

	err := app.Run([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApp_RunNoArgs(t *testing.T) {
	out := &bytes.Buffer{}
	router := NewRouter(out, out, nil)
	app := &App{router: router}

	err := app.Run([]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "go-migration") {
		t.Errorf("expected help output to contain 'go-migration', got %q", out.String())
	}
}

func TestApp_RunConfigFlag(t *testing.T) {
	out := &bytes.Buffer{}
	router := NewRouter(out, out, nil)
	app := &App{router: router}

	var capturedConfig string
	router.Register("test", "test cmd", func(args []string, flags *flag.FlagSet) error {
		return nil
	}, nil)
	router.SetPreRun(func(cmdName, configPath string, flags *flag.FlagSet) error {
		capturedConfig = configPath
		return nil
	})

	err := app.Run([]string{"--config", "custom.yaml", "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if capturedConfig != "custom.yaml" {
		t.Errorf("got config %q, want %q", capturedConfig, "custom.yaml")
	}
}
