package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AcquireAdvisoryLock acquires Rift's PostgreSQL advisory lock for migration execution.
func AcquireAdvisoryLock(ctx context.Context, pool *pgxpool.Pool) (func(), error) {
	if pool == nil {
		return nil, fmt.Errorf("acquiring migration advisory lock: database pool is nil")
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection for migration advisory lock: %w", err)
	}

	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtext('rift_migration_lock'))").Scan(&acquired); err != nil {
		conn.Release()
		return nil, fmt.Errorf("trying migration advisory lock: %w", err)
	}
	if !acquired {
		conn.Release()
		return nil, fmt.Errorf("another rift migration is already in progress; try again shortly")
	}

	release := func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock(hashtext('rift_migration_lock'))")
		conn.Release()
	}
	return release, nil
}

// RunUp applies all pending local migrations in order and records each success.
func RunUp(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, dryRun bool) error {
	if cfg == nil {
		return fmt.Errorf("running migrations up: config is required")
	}
	if pool == nil {
		return fmt.Errorf("running migrations up: database pool is nil")
	}

	release, err := AcquireAdvisoryLock(ctx, pool)
	if err != nil {
		return err
	}
	defer release()

	if err := EnsureStateTable(ctx, pool); err != nil {
		return err
	}

	files, err := LoadFiles(cfg.MigrationsDir)
	if err != nil {
		return err
	}
	applied, err := GetApplied(ctx, pool)
	if err != nil {
		return err
	}
	if conflicts := DetectConflicts(applied, files); len(conflicts) > 0 {
		return fmt.Errorf("migration conflicts detected: %d conflict(s) must be resolved before applying", len(conflicts))
	}

	appliedByVersion := make(map[string]MigrationRecord, len(applied))
	for _, record := range applied {
		if !record.RolledBack {
			appliedByVersion[record.Version] = record
		}
	}

	for _, file := range files {
		if _, ok := appliedByVersion[file.Version]; ok {
			continue
		}
		if dryRun {
			continue
		}

		startedAt := time.Now()
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning migration %s transaction: %w", file.Version, err)
		}

		if _, err := tx.Exec(ctx, file.UpSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("applying migration %s: %w", file.Version, err)
		}
		if err := RecordApplied(ctx, tx, MigrationRecord{
			Version:     file.Version,
			Filename:    file.Filename,
			Checksum:    file.Checksum,
			AppliedBy:   cfg.Author,
			ExecutionMs: int(time.Since(startedAt).Milliseconds()),
		}); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing migration %s transaction: %w", file.Version, err)
		}
	}

	return nil
}

// RunDown rolls back the most recently applied migrations up to steps.
func RunDown(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, steps int) error {
	if cfg == nil {
		return fmt.Errorf("running migrations down: config is required")
	}
	if pool == nil {
		return fmt.Errorf("running migrations down: database pool is nil")
	}
	if steps <= 0 {
		return fmt.Errorf("running migrations down: steps must be greater than zero")
	}

	release, err := AcquireAdvisoryLock(ctx, pool)
	if err != nil {
		return err
	}
	defer release()

	files, err := LoadFiles(cfg.MigrationsDir)
	if err != nil {
		return err
	}
	filesByVersion := make(map[string]MigrationFile, len(files))
	for _, file := range files {
		filesByVersion[file.Version] = file
	}

	applied, err := GetApplied(ctx, pool)
	if err != nil {
		return err
	}

	rolledBack := 0
	for i := len(applied) - 1; i >= 0 && rolledBack < steps; i-- {
		record := applied[i]
		if record.RolledBack {
			continue
		}

		file, ok := filesByVersion[record.Version]
		if !ok {
			return fmt.Errorf("rolling back migration %s: local down migration file is missing", record.Version)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning rollback %s transaction: %w", record.Version, err)
		}
		if _, err := tx.Exec(ctx, file.DownSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("rolling back migration %s: %w", record.Version, err)
		}
		if err := RecordRolledBack(ctx, tx, record.Version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing rollback %s transaction: %w", record.Version, err)
		}
		rolledBack++
	}

	return nil
}
