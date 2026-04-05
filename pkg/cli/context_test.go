package cli

import (
	"testing"

	"github.com/gopackx/go-migration/internal/generator"
	"github.com/gopackx/go-migration/internal/logger"
	"github.com/gopackx/go-migration/pkg/config"
)

func TestNewCLIContext(t *testing.T) {
	cfg := &config.Config{DefaultConn: "test"}
	gen := generator.NewGenerator("output")
	log := logger.NopLogger{}

	ctx := NewCLIContext(cfg, nil, nil, nil, gen, log)

	if ctx == nil {
		t.Fatalf("expected non-nil")
	}
	if ctx.Config != cfg {
		t.Errorf("got %v, want %v", ctx.Config, cfg)
	}
	if ctx.DB != nil {
		t.Errorf("expected nil, got %v", ctx.DB)
	}
	if ctx.Migrator != nil {
		t.Errorf("expected nil, got %v", ctx.Migrator)
	}
	if ctx.Seeder != nil {
		t.Errorf("expected nil, got %v", ctx.Seeder)
	}
	if ctx.Generator != gen {
		t.Errorf("got %v, want %v", ctx.Generator, gen)
	}
	if ctx.Logger != log {
		t.Errorf("got %v, want %v", ctx.Logger, log)
	}
}

func TestNewCLIContextAllNil(t *testing.T) {
	ctx := NewCLIContext(nil, nil, nil, nil, nil, nil)

	if ctx == nil {
		t.Fatalf("expected non-nil")
	}
	if ctx.Config != nil {
		t.Errorf("expected nil, got %v", ctx.Config)
	}
	if ctx.DB != nil {
		t.Errorf("expected nil, got %v", ctx.DB)
	}
	if ctx.Migrator != nil {
		t.Errorf("expected nil, got %v", ctx.Migrator)
	}
	if ctx.Seeder != nil {
		t.Errorf("expected nil, got %v", ctx.Seeder)
	}
	if ctx.Generator != nil {
		t.Errorf("expected non-nil")
	}
	if ctx.Logger != nil {
		t.Errorf("expected nil, got %v", ctx.Logger)
	}
}
