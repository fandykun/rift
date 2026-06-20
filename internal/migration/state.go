package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationRecord is one row from Rift's PostgreSQL migration state table.
type MigrationRecord struct {
	ID          int
	Version     string
	Filename    string
	Checksum    string
	AppliedAt   time.Time
	AppliedBy   string
	ExecutionMs int
	RolledBack  bool
}

// EnsureStateTable creates Rift's migration state table when it does not exist.
func EnsureStateTable(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("ensuring migration state table: database pool is nil")
	}

	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS _rift_migrations (
  id            SERIAL PRIMARY KEY,
  version       TEXT NOT NULL UNIQUE,
  filename      TEXT NOT NULL,
  checksum      TEXT NOT NULL,
  applied_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  applied_by    TEXT,
  execution_ms  INTEGER,
  rolled_back   BOOLEAN NOT NULL DEFAULT false
);
`)
	if err != nil {
		return fmt.Errorf("ensuring migration state table: %w", err)
	}
	return nil
}

// GetApplied returns migration records ordered by application time and ID.
func GetApplied(ctx context.Context, pool *pgxpool.Pool) ([]MigrationRecord, error) {
	if pool == nil {
		return nil, fmt.Errorf("getting applied migrations: database pool is nil")
	}

	rows, err := pool.Query(ctx, `
SELECT id, version, filename, checksum, applied_at, COALESCE(applied_by, ''), COALESCE(execution_ms, 0), rolled_back
FROM _rift_migrations
ORDER BY applied_at ASC, id ASC;
`)
	if err != nil {
		return nil, fmt.Errorf("querying applied migrations: %w", err)
	}
	defer rows.Close()

	records, err := pgx.CollectRows(rows, pgx.RowToStructByPos[MigrationRecord])
	if err != nil {
		return nil, fmt.Errorf("scanning applied migrations: %w", err)
	}
	return records, nil
}

// RecordApplied inserts or updates the state row for a successfully applied migration.
func RecordApplied(ctx context.Context, tx pgx.Tx, record MigrationRecord) error {
	if tx == nil {
		return fmt.Errorf("recording applied migration %q: transaction is nil", record.Version)
	}
	if record.Version == "" {
		return fmt.Errorf("recording applied migration: version is required")
	}
	if record.Filename == "" {
		return fmt.Errorf("recording applied migration %q: filename is required", record.Version)
	}
	if record.Checksum == "" {
		return fmt.Errorf("recording applied migration %q: checksum is required", record.Version)
	}

	_, err := tx.Exec(ctx, `
INSERT INTO _rift_migrations (version, filename, checksum, applied_by, execution_ms, rolled_back)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, false)
ON CONFLICT (version) DO UPDATE SET
  filename = EXCLUDED.filename,
  checksum = EXCLUDED.checksum,
  applied_at = now(),
  applied_by = EXCLUDED.applied_by,
  execution_ms = EXCLUDED.execution_ms,
  rolled_back = false;
`, record.Version, record.Filename, record.Checksum, record.AppliedBy, record.ExecutionMs)
	if err != nil {
		return fmt.Errorf("recording applied migration %q: %w", record.Version, err)
	}
	return nil
}

// RecordRolledBack marks a migration version as rolled back.
func RecordRolledBack(ctx context.Context, tx pgx.Tx, version string) error {
	if tx == nil {
		return fmt.Errorf("recording rolled back migration %q: transaction is nil", version)
	}
	if version == "" {
		return fmt.Errorf("recording rolled back migration: version is required")
	}

	commandTag, err := tx.Exec(ctx, `
UPDATE _rift_migrations
SET rolled_back = true
WHERE version = $1;
`, version)
	if err != nil {
		return fmt.Errorf("recording rolled back migration %q: %w", version, err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("recording rolled back migration %q: migration record not found", version)
	}
	return nil
}
