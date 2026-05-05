package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure no env vars interfere
	os.Unsetenv("BRIDGE_CONTEXT")
	os.Unsetenv("BRIDGE_SERVER")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.CurrentContext != "local" {
		t.Errorf("expected current context 'local', got '%s'", cfg.CurrentContext)
	}

	ctx := cfg.ActiveContext()
	if ctx.Server != "http://localhost:8082" {
		t.Errorf("expected server 'http://localhost:8082', got '%s'", ctx.Server)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("BRIDGE_SERVER", "http://overridden:8082")
	defer os.Unsetenv("BRIDGE_SERVER")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	ctx := cfg.ActiveContext()
	if ctx.Server != "http://overridden:8082" {
		t.Errorf("expected server 'http://overridden:8082', got '%s'", ctx.Server)
	}
}
