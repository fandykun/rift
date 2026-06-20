package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesDefaultsWhenFileMissing(t *testing.T) {
	t.Setenv("RIFT_DATABASE_URL", "")
	t.Setenv("RIFT_ENV", "")
	t.Setenv("RIFT_MIGRATIONS_DIR", "")
	t.Setenv("RIFT_PORT", "")
	t.Setenv("RIFT_TOKEN", "")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Environment != DefaultEnvironment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, DefaultEnvironment)
	}
	if cfg.MigrationsDir != DefaultMigrationsDir {
		t.Fatalf("MigrationsDir = %q, want %q", cfg.MigrationsDir, DefaultMigrationsDir)
	}
	if cfg.Server.Port != DefaultPort {
		t.Fatalf("Server.Port = %d, want %d", cfg.Server.Port, DefaultPort)
	}
}

func TestLoadYAMLAndEnvironmentPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rift.yaml")
	content := []byte(`environment: staging
database_url: yaml-database-url
migrations_dir: ./yaml-migrations
author: yaml-author
server:
  port: 7879
  token: yaml-token
linter:
  warn_only: true
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("writing config fixture: %v", err)
	}

	t.Setenv("RIFT_DATABASE_URL", "env-database-url")
	t.Setenv("RIFT_ENV", "production")
	t.Setenv("RIFT_MIGRATIONS_DIR", "./env-migrations")
	t.Setenv("RIFT_PORT", "9000")
	t.Setenv("RIFT_TOKEN", "env-token")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.DatabaseURL != "env-database-url" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.Environment != "production" {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
	if cfg.MigrationsDir != "./env-migrations" {
		t.Fatalf("MigrationsDir = %q", cfg.MigrationsDir)
	}
	if cfg.Server.Port != 9000 {
		t.Fatalf("Server.Port = %d", cfg.Server.Port)
	}
	if cfg.Server.Token != "env-token" {
		t.Fatalf("Server.Token = %q", cfg.Server.Token)
	}
	if cfg.Author != "yaml-author" {
		t.Fatalf("Author = %q", cfg.Author)
	}
	if !cfg.Linter.WarnOnly {
		t.Fatalf("Linter.WarnOnly = false, want true")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("RIFT_PORT", "not-a-port")

	_, err := Load("")
	if err == nil {
		t.Fatal("Load returned nil error for invalid RIFT_PORT")
	}
}
