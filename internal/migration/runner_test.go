package migration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAcquireAdvisoryLockIntegration(t *testing.T) {
	pool := integrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	release, err := AcquireAdvisoryLock(ctx, pool)
	if err != nil {
		t.Fatalf("AcquireAdvisoryLock() error = %v", err)
	}
	defer release()

	_, err = AcquireAdvisoryLock(ctx, pool)
	if err == nil {
		t.Fatal("expected second advisory lock acquisition to fail")
	}
}

func TestRunUpAndRunDownIntegration(t *testing.T) {
	pool := integrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS posts; DROP TABLE IF EXISTS users; DROP TABLE IF EXISTS _rift_migrations;`); err != nil {
		t.Fatalf("resetting integration database: %v", err)
	}

	migrationsDir := t.TempDir()
	writeMigrationFile(t, migrationsDir, "20260620_120000_create_users.up.sql", `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);`)
	writeMigrationFile(t, migrationsDir, "20260620_120000_create_users.down.sql", `DROP TABLE users;`)
	writeMigrationFile(t, migrationsDir, "20260620_120001_create_posts.up.sql", `CREATE TABLE posts (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id));`)
	writeMigrationFile(t, migrationsDir, "20260620_120001_create_posts.down.sql", `DROP TABLE posts;`)
	writeMigrationFile(t, migrationsDir, "20260620_120002_add_email.up.sql", `ALTER TABLE users ADD COLUMN email TEXT;`)
	writeMigrationFile(t, migrationsDir, "20260620_120002_add_email.down.sql", `ALTER TABLE users DROP COLUMN email;`)

	cfg := &config.Config{MigrationsDir: migrationsDir, Author: "rift-test"}
	if err := RunUp(ctx, pool, cfg, false); err != nil {
		t.Fatalf("RunUp() error = %v", err)
	}

	records, err := GetApplied(ctx, pool)
	if err != nil {
		t.Fatalf("GetApplied() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 applied records, got %d", len(records))
	}

	if err := RunDown(ctx, pool, cfg, 1); err != nil {
		t.Fatalf("RunDown() error = %v", err)
	}

	records, err = GetApplied(ctx, pool)
	if err != nil {
		t.Fatalf("GetApplied() after rollback error = %v", err)
	}
	if !records[2].RolledBack {
		t.Fatalf("expected latest migration to be marked rolled back")
	}

	var emailColumnCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'users'
  AND column_name = 'email';
`).Scan(&emailColumnCount); err != nil {
		t.Fatalf("checking rolled back column: %v", err)
	}
	if emailColumnCount != 0 {
		t.Fatalf("expected users.email to be rolled back, found %d column(s)", emailColumnCount)
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("RIFT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RIFT_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating integration pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
