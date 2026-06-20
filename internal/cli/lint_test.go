package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
)

func TestRunLintSpecificFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "dangerous.up.sql")
	if err := os.WriteFile(filePath, []byte("ALTER TABLE users DROP COLUMN legacy;"), 0o600); err != nil {
		t.Fatalf("writing lint target: %v", err)
	}

	var stdout bytes.Buffer
	err := RunLint(context.Background(), &stdout, filepath.Join(tmpDir, "missing-rift.yaml"), filePath)
	if !errors.Is(err, ErrLintErrorsFound) {
		t.Fatalf("expected ErrLintErrorsFound, got %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "DROP_COLUMN") || !strings.Contains(output, "dangerous.up.sql:1") {
		t.Fatalf("expected lint output with DROP_COLUMN and file line, got %q", output)
	}
}

func TestRunLintWarnOnlyAllowsErrors(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "dangerous.up.sql")
	if err := os.WriteFile(filePath, []byte("DROP TABLE users;"), 0o600); err != nil {
		t.Fatalf("writing lint target: %v", err)
	}
	configPath := filepath.Join(tmpDir, "rift.yaml")
	if err := os.WriteFile(configPath, []byte("linter:\n  warn_only: true\n"), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	var stdout bytes.Buffer
	if err := RunLint(context.Background(), &stdout, configPath, filePath); err != nil {
		t.Fatalf("RunLint() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "DROP_TABLE") {
		t.Fatalf("expected lint output to include DROP_TABLE, got %q", stdout.String())
	}
}

func TestRunLintPendingMigrationsIntegration(t *testing.T) {
	configPath := setupLintIntegration(t)

	var stdout bytes.Buffer
	if err := RunLint(context.Background(), &stdout, configPath, ""); err != nil {
		t.Fatalf("RunLint() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "CREATE_INDEX_WITHOUT_CONCURRENTLY") || !strings.Contains(output, "20260620_150000_add_index") {
		t.Fatalf("expected pending migration lint output, got %q", output)
	}
}

func setupLintIntegration(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("RIFT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RIFT_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating setup pool: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS users CASCADE; DROP TABLE IF EXISTS _rift_migrations CASCADE;`); err != nil {
		t.Fatalf("resetting database: %v", err)
	}
	if err := migration.EnsureStateTable(ctx, pool); err != nil {
		t.Fatalf("ensuring state table: %v", err)
	}

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	writeMigrationFile(t, migrationsDir, "20260620_150000_add_index.up.sql", "CREATE INDEX users_email_idx ON users (email);")
	writeMigrationFile(t, migrationsDir, "20260620_150000_add_index.down.sql", "DROP INDEX IF EXISTS users_email_idx;")

	configPath := filepath.Join(tmpDir, "rift.yaml")
	content := "database_url: " + databaseURL + "\nmigrations_dir: " + migrationsDir + "\nauthor: lint-test\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return configPath
}
