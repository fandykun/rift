package migration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
)

func TestRecordAppliedValidation(t *testing.T) {
	ctx := context.Background()

	if err := RecordApplied(ctx, nil, MigrationRecord{}); err == nil {
		t.Fatal("expected nil transaction error")
	}
}

func TestEnsureStateTableAndRecordsIntegration(t *testing.T) {
	databaseURL := os.Getenv("RIFT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RIFT_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS _rift_migrations`); err != nil {
		t.Fatalf("resetting state table: %v", err)
	}

	if err := EnsureStateTable(ctx, pool); err != nil {
		t.Fatalf("ensuring state table first time: %v", err)
	}
	if err := EnsureStateTable(ctx, pool); err != nil {
		t.Fatalf("ensuring state table second time: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning apply transaction: %v", err)
	}
	if err := RecordApplied(ctx, tx, MigrationRecord{
		Version:     "20260620_120000",
		Filename:    "20260620_120000_create_users.up.sql",
		Checksum:    "abc123",
		AppliedBy:   "rift-test",
		ExecutionMs: 42,
	}); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording applied migration: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("committing apply transaction: %v", err)
	}

	records, err := GetApplied(ctx, pool)
	if err != nil {
		t.Fatalf("getting applied migrations: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 migration record, got %d", len(records))
	}
	if records[0].Version != "20260620_120000" {
		t.Fatalf("unexpected version: %s", records[0].Version)
	}
	if records[0].RolledBack {
		t.Fatal("newly applied migration should not be marked rolled back")
	}
	if records[0].ExecutionMs != 42 {
		t.Fatalf("unexpected execution_ms: %d", records[0].ExecutionMs)
	}

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning rollback transaction: %v", err)
	}
	if err := RecordRolledBack(ctx, tx, "20260620_120000"); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording rolled back migration: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("committing rollback transaction: %v", err)
	}

	records, err = GetApplied(ctx, pool)
	if err != nil {
		t.Fatalf("getting rolled back migrations: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 migration record after rollback, got %d", len(records))
	}
	if !records[0].RolledBack {
		t.Fatal("expected migration to be marked rolled back")
	}
}
