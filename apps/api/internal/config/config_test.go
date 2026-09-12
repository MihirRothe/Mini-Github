package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.GitDefaultBranch != "main" {
		t.Errorf("expected default branch 'main', got %s", cfg.GitDefaultBranch)
	}
	if cfg.IsProduction() {
		t.Errorf("expected development env by default, got production")
	}
	if cfg.Addr() != "0.0.0.0:8080" {
		t.Errorf("expected addr '0.0.0.0:8080', got %s", cfg.Addr())
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("FORGEHUB_PORT", "9000")
	os.Setenv("FORGEHUB_ENV", "production")
	defer func() {
		os.Unsetenv("FORGEHUB_PORT")
		os.Unsetenv("FORGEHUB_ENV")
	}()

	cfg := Load()
	if cfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Port)
	}
	if !cfg.IsProduction() {
		t.Errorf("expected production env")
	}
}
