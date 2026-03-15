package config

import (
	"os"
	"testing"
)

func TestLoadDefaultsAndRequiredJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.MigrationsPath == "" {
		t.Fatal("expected default migrations path")
	}
}

func TestLoadMissingJWTSecret(t *testing.T) {
	// force unset even if globally set
	_ = os.Unsetenv("JWT_SECRET")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET missing")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("PORT", "9999")
	t.Setenv("CORS_ORIGINS", "http://a.local, http://b.local")
	t.Setenv("MIGRATIONS_PATH", "file:///tmp/migrations")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != "9999" {
		t.Fatalf("expected port override, got %s", cfg.Port)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("expected 2 cors origins, got %d", len(cfg.CORSOrigins))
	}
	if cfg.MigrationsPath != "file:///tmp/migrations" {
		t.Fatalf("expected migrations override, got %s", cfg.MigrationsPath)
	}
}
