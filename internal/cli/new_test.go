package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunNewCreatesMigrationPair(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rift.yaml")
	migrationsDir := filepath.Join(tmpDir, "migrations")
	writeConfig(t, configPath, migrationsDir)

	createdAt := time.Date(2026, 6, 20, 12, 34, 56, 0, time.UTC)
	var stdout bytes.Buffer
	if err := RunNew(&stdout, configPath, "Add Users Table", createdAt); err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}

	upPath := filepath.Join(migrationsDir, "20260620_123456_add_users_table.up.sql")
	downPath := filepath.Join(migrationsDir, "20260620_123456_add_users_table.down.sql")
	assertFileContains(t, upPath, "-- Migration: add_users_table | Created: 2026-06-20T12:34:56Z")
	assertFileContains(t, downPath, "-- Write your rollback migration SQL here.")

	output := stdout.String()
	if !strings.Contains(output, upPath) || !strings.Contains(output, downPath) {
		t.Fatalf("expected output to include created paths, got %q", output)
	}
}

func TestRunNewRejectsEmptyNormalizedName(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rift.yaml")
	writeConfig(t, configPath, filepath.Join(tmpDir, "migrations"))

	var stdout bytes.Buffer
	if err := RunNew(&stdout, configPath, "---", time.Now()); err == nil {
		t.Fatal("expected empty normalized migration name error")
	}
}

func TestRunNewDoesNotOverwriteExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rift.yaml")
	migrationsDir := filepath.Join(tmpDir, "migrations")
	writeConfig(t, configPath, migrationsDir)
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	existingPath := filepath.Join(migrationsDir, "20260620_123456_add_users_table.up.sql")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("writing existing migration: %v", err)
	}

	createdAt := time.Date(2026, 6, 20, 12, 34, 56, 0, time.UTC)
	var stdout bytes.Buffer
	if err := RunNew(&stdout, configPath, "add users table", createdAt); err == nil {
		t.Fatal("expected existing file error")
	}
}

func writeConfig(t *testing.T, configPath string, migrationsDir string) {
	t.Helper()
	content := "migrations_dir: " + migrationsDir + "\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("expected %s to contain %q, got %q", path, want, string(content))
	}
}
