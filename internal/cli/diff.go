package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	internaldiff "github.com/fandykun/rift/internal/diff"
	"github.com/fandykun/rift/internal/migration"
	"github.com/spf13/cobra"
)

// ErrSchemaDiffDetected is returned by rift diff when schema changes are present.
var ErrSchemaDiffDetected = errors.New("schema diff detected")

// NewDiffCommand returns the `rift diff` command for previewing pending schema changes.
func NewDiffCommand(configPath *string) *cobra.Command {
	var outputJSON bool
	var schema string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare pending migrations against the live database schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDiff(cmd.Context(), cmd.OutOrStdout(), *configPath, schema, outputJSON)
		},
	}
	cmd.Flags().BoolVar(&outputJSON, "json", false, "render schema diff as JSON")
	cmd.Flags().StringVar(&schema, "schema", "public", "PostgreSQL schema to introspect")
	return cmd
}

// RunDiff computes and renders the diff introduced by pending migrations.
func RunDiff(ctx context.Context, stdout io.Writer, configPath string, schema string, outputJSON bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(schema) == "" {
		return fmt.Errorf("schema is required")
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(checkCtx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Ping(checkCtx, pool); err != nil {
		return err
	}
	if err := migration.EnsureStateTable(checkCtx, pool); err != nil {
		return err
	}

	files, err := migration.LoadFiles(cfg.MigrationsDir)
	if err != nil {
		return err
	}
	applied, err := migration.GetApplied(checkCtx, pool)
	if err != nil {
		return err
	}

	liveSnapshot, err := internaldiff.IntrospectLive(checkCtx, pool, schema)
	if err != nil {
		return err
	}

	expectedSnapshot := liveSnapshot
	for _, file := range pendingMigrations(applied, files) {
		expectedSnapshot, err = internaldiff.ApplyMigrationSQL(expectedSnapshot, file.UpSQL)
		if err != nil {
			return fmt.Errorf("applying pending migration %q to expected schema: %w", file.Filename, err)
		}
	}

	schemaDiff := internaldiff.ComputeDiff(liveSnapshot, expectedSnapshot)
	if outputJSON {
		content, err := internaldiff.RenderJSON(schemaDiff)
		if err != nil {
			return err
		}
		if _, err := stdout.Write(content); err != nil {
			return fmt.Errorf("writing JSON diff output: %w", err)
		}
	} else if err := internaldiff.RenderTerminal(stdout, schemaDiff); err != nil {
		return err
	}

	if internaldiff.HasChanges(schemaDiff) {
		return ErrSchemaDiffDetected
	}
	return nil
}
