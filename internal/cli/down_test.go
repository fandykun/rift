package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
)

func TestRunDownAbortDoesNotRequireDatabase(t *testing.T) {
	var stdout bytes.Buffer
	if err := RunDown(context.Background(), strings.NewReader("no\n"), &stdout, "missing.yaml", 1); err != nil {
		t.Fatalf("RunDown(abort) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Rollback aborted") {
		t.Fatalf("expected abort output, got %q", stdout.String())
	}
}

func TestRunDownIntegration(t *testing.T) {
	configPath, databaseURL := setupUpIntegration(t)
	var upOutput bytes.Buffer
	if err := RunUp(context.Background(), &upOutput, configPath, false, false); err != nil {
		t.Fatalf("RunUp() setup error = %v", err)
	}

	var stdout bytes.Buffer
	if err := RunDown(context.Background(), strings.NewReader("yes\n"), &stdout, configPath, 1); err != nil {
		t.Fatalf("RunDown() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "OK rolled back 1 migration") {
		t.Fatalf("expected rollback output, got %q", stdout.String())
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
		t.Fatalf("expected 2 migration records, got %d", len(records))
	}
	if !records[1].RolledBack {
		t.Fatal("expected latest migration to be marked rolled back")
	}

	var postsTableCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = current_schema()
  AND table_name = 'posts';
`).Scan(&postsTableCount); err != nil {
		t.Fatalf("checking rolled back table: %v", err)
	}
	if postsTableCount != 0 {
		t.Fatalf("expected posts table to be rolled back, found %d table(s)", postsTableCount)
	}
}

func TestRunDownRejectsInvalidSteps(t *testing.T) {
	if err := RunDown(context.Background(), strings.NewReader("yes\n"), bytes.NewBuffer(nil), "missing.yaml", 0); err == nil {
		t.Fatal("expected invalid steps error")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
