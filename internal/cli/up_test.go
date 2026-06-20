package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
)

func TestPendingMigrations(t *testing.T) {
	files := []migration.MigrationFile{
		{Version: "20260620_120000", Filename: "20260620_120000_create_users"},
		{Version: "20260620_120001", Filename: "20260620_120001_create_posts"},
	}
	applied := []migration.MigrationRecord{{Version: "20260620_120000", RolledBack: false}}

	pending := pendingMigrations(applied, files)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending migration, got %d", len(pending))
	}
	if pending[0].Version != "20260620_120001" {
		t.Fatalf("unexpected pending version: %s", pending[0].Version)
	}
}

func TestRunUpDryRunIntegration(t *testing.T) {
	configPath, _ := setupUpIntegration(t)

	var stdout bytes.Buffer
	if err := RunUp(context.Background(), &stdout, configPath, true, false); err != nil {
		t.Fatalf("RunUp(dryRun) error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "DRY-RUN 2 pending migration(s)") {
		t.Fatalf("expected dry-run output to list pending migrations, got %q", output)
	}
	if !strings.Contains(output, "20260620_120000_create_users") {
		t.Fatalf("expected dry-run output to include migration filename, got %q", output)
	}
}

func TestRunUpIntegration(t *testing.T) {
	configPath, databaseURL := setupUpIntegration(t)

	var stdout bytes.Buffer
	if err := RunUp(context.Background(), &stdout, configPath, false, false); err != nil {
		t.Fatalf("RunUp() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "OK applied 20260620_120000_create_users") {
		t.Fatalf("expected apply output, got %q", output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating verification pool: %v", err)
	}
	defer pool.Close()

	records, err := migration.GetApplied(ctx, pool)
	if err != nil {
		t.Fatalf("getting applied migrations: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 applied migration records, got %d", len(records))
	}
}

func writeMigrationFile(t *testing.T, dir string, name string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("writing migration file %s: %v", name, err)
	}
}

func TestRunUpConflictRequiresForceIntegration(t *testing.T) {
	configPath, databaseURL := setupUpIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating setup pool: %v", err)
	}
	defer pool.Close()
	if err := migration.EnsureStateTable(ctx, pool); err != nil {
		t.Fatalf("ensuring state table: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning conflict setup transaction: %v", err)
	}
	if err := migration.RecordApplied(ctx, tx, migration.MigrationRecord{Version: "20260620_115959", Filename: "20260620_115959_missing", Checksum: "missing-checksum"}); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording missing-file conflict: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("committing conflict setup transaction: %v", err)
	}

	var stdout bytes.Buffer
	if err := RunUp(context.Background(), &stdout, configPath, false, false); err == nil {
		t.Fatal("expected conflict error without force")
	}

	stdout.Reset()
	if err := RunUp(context.Background(), &stdout, configPath, false, true); err != nil {
		t.Fatalf("expected --force to apply pending migrations despite conflict, got %v", err)
	}
	if !strings.Contains(stdout.String(), "WARN continuing despite 1 conflict") {
		t.Fatalf("expected force warning, got %q", stdout.String())
	}
}

func setupUpIntegration(t *testing.T) (string, string) {
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
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS posts CASCADE; DROP TABLE IF EXISTS users CASCADE; DROP TABLE IF EXISTS _rift_migrations CASCADE;`); err != nil {
		t.Fatalf("resetting integration database: %v", err)
	}

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	writeMigrationFile(t, migrationsDir, "20260620_120000_create_users.up.sql", `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);`)
	writeMigrationFile(t, migrationsDir, "20260620_120000_create_users.down.sql", `DROP TABLE users;`)
	writeMigrationFile(t, migrationsDir, "20260620_120001_create_posts.up.sql", `CREATE TABLE posts (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id));`)
	writeMigrationFile(t, migrationsDir, "20260620_120001_create_posts.down.sql", `DROP TABLE posts;`)

	configPath := filepath.Join(tmpDir, "rift.yaml")
	content := "database_url: " + databaseURL + "\nmigrations_dir: " + migrationsDir + "\nauthor: cli-test\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return configPath, databaseURL
}
