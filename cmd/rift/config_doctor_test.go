package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fandykun/rift/internal/config"
)

func TestConfigDoctorPassesForPublicReadyConfigWhenSkippingDB(t *testing.T) {
	migrationsDir := t.TempDir()
	cfg := &config.Config{
		Environment:   "demo",
		DatabaseURL:   "postgres://rift:password@example.internal:5432/rift?sslmode=require",
		MigrationsDir: migrationsDir,
		Server: config.ServerConfig{
			Port:  7878,
			Token: "1234567890abcdef1234567890abcdef",
		},
	}

	checks := evaluateConfigDoctor(context.Background(), cfg, true)
	for _, check := range checks {
		if check.Severity == doctorFail {
			t.Fatalf("unexpected failure for %s: %s", check.Name, check.Message)
		}
	}
}

func TestConfigDoctorFailsUnsafeProductionToken(t *testing.T) {
	cfg := &config.Config{
		Environment:   "production",
		DatabaseURL:   "postgres://rift:password@example.internal:5432/rift",
		MigrationsDir: t.TempDir(),
		Server: config.ServerConfig{
			Port:  7878,
			Token: "local-dev-token",
		},
	}

	checks := evaluateConfigDoctor(context.Background(), cfg, true)
	if !hasDoctorFailure(checks, "api token") {
		t.Fatalf("expected api token failure, got %+v", checks)
	}
}

func TestConfigDoctorFailsMissingMigrationsDir(t *testing.T) {
	cfg := &config.Config{
		Environment:   "demo",
		DatabaseURL:   "postgres://rift:password@example.internal:5432/rift",
		MigrationsDir: filepath.Join(t.TempDir(), "missing"),
		Server: config.ServerConfig{
			Port:  7878,
			Token: "1234567890abcdef1234567890abcdef",
		},
	}

	checks := evaluateConfigDoctor(context.Background(), cfg, true)
	if !hasDoctorFailure(checks, "migrations dir") {
		t.Fatalf("expected migrations dir failure, got %+v", checks)
	}
}

func TestConfigDoctorRedactsDatabasePassword(t *testing.T) {
	redacted := redactDatabaseURL("postgres://rift:password@example.internal:5432/rift?sslmode=require")
	if strings.Contains(redacted, "password") {
		t.Fatalf("redacted URL leaked password: %s", redacted)
	}
	if !strings.Contains(redacted, "rift:%5Bredacted%5D@example.internal") {
		t.Fatalf("redacted URL did not preserve useful connection context: %s", redacted)
	}
}

func hasDoctorFailure(checks []doctorCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name && check.Severity == doctorFail {
			return true
		}
	}
	return false
}
