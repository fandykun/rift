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
)

func TestRunDiffNoPendingIntegration(t *testing.T) {
	configPath, _ := setupDiffIntegration(t, nil)

	var stdout bytes.Buffer
	if err := RunDiff(context.Background(), &stdout, configPath, "public", false); err != nil {
		t.Fatalf("RunDiff() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "OK no schema changes") {
		t.Fatalf("expected no changes output, got %q", stdout.String())
	}
}

func TestRunDiffPendingMigrationIntegration(t *testing.T) {
	configPath, _ := setupDiffIntegration(t, map[string]string{
		"20260620_130000_add_user_email.up.sql":   `ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''; CREATE INDEX users_email_idx ON users (email);`,
		"20260620_130000_add_user_email.down.sql": `ALTER TABLE users DROP COLUMN email; DROP INDEX IF EXISTS users_email_idx;`,
	})

	var stdout bytes.Buffer
	err := RunDiff(context.Background(), &stdout, configPath, "public", false)
	if !errors.Is(err, ErrSchemaDiffDetected) {
		t.Fatalf("expected ErrSchemaDiffDetected, got %v", err)
	}
	output := stdout.String()
	for _, expected := range []string{"DIFF", "1 table modified", "1 column added", "1 index changed", "users", "email", "users_email_idx"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected diff output to contain %q, got %q", expected, output)
		}
	}
}

func TestRunDiffJSONIntegration(t *testing.T) {
	configPath, _ := setupDiffIntegration(t, map[string]string{
		"20260620_130000_create_projects.up.sql":   `CREATE TABLE projects (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);`,
		"20260620_130000_create_projects.down.sql": `DROP TABLE projects;`,
	})

	var stdout bytes.Buffer
	err := RunDiff(context.Background(), &stdout, configPath, "public", true)
	if !errors.Is(err, ErrSchemaDiffDetected) {
		t.Fatalf("expected ErrSchemaDiffDetected, got %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"TablesAdded"`) || !strings.Contains(output, `"projects"`) {
		t.Fatalf("expected JSON diff output, got %q", output)
	}
}

func setupDiffIntegration(t *testing.T, files map[string]string) (string, string) {
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
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS projects CASCADE; DROP TABLE IF EXISTS posts CASCADE; DROP TABLE IF EXISTS users CASCADE; DROP TABLE IF EXISTS _rift_migrations CASCADE;`); err != nil {
		t.Fatalf("resetting integration database: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);`); err != nil {
		t.Fatalf("creating live schema: %v", err)
	}

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	for name, content := range files {
		writeMigrationFile(t, migrationsDir, name, content)
	}

	configPath := filepath.Join(tmpDir, "rift.yaml")
	content := "database_url: " + databaseURL + "\nmigrations_dir: " + migrationsDir + "\nauthor: cli-test\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return configPath, databaseURL
}
