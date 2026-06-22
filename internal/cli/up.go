package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewUpCommand returns the `rift up` command for applying pending migrations.
func NewUpCommand(configPath *string) *cobra.Command {
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunUp(cmd.Context(), cmd.OutOrStdout(), *configPath, dryRun, force)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview pending migrations without applying them")
	cmd.Flags().BoolVar(&force, "force", false, "continue despite migration conflicts or linter errors where supported")
	return cmd
}

// RunUp applies pending migrations after loading config and verifying database connectivity.
func RunUp(ctx context.Context, stdout io.Writer, configPath string, dryRun bool, force bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
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
	conflicts := migration.DetectConflicts(applied, files)
	if len(conflicts) > 0 && !force {
		return fmt.Errorf("migration conflicts detected: %d conflict(s); resolve conflicts or rerun with --force after manual review", len(conflicts))
	}

	pending := pendingMigrations(applied, files)
	lintResults := lintMigrationFiles(pending)
	lintErrors, lintWarnings := lintCounts(lintResults)
	ok := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()

	if len(pending) == 0 {
		fmt.Fprintf(stdout, "%s no pending migrations\n", ok("OK"))
		return nil
	}

	if dryRun {
		fmt.Fprintf(stdout, "%s %d pending migration(s):\n", warn("DRY-RUN"), len(pending))
		for _, file := range pending {
			fmt.Fprintf(stdout, "- %s\n", file.Filename)
		}
		if lintErrors > 0 || lintWarnings > 0 {
			return renderLintResults(stdout, lintResults, lintErrors, lintWarnings)
		}
		return nil
	}

	if lintErrors > 0 && !force && !cfg.Linter.WarnOnly {
		if err := renderLintResults(stdout, lintResults, lintErrors, lintWarnings); err != nil {
			return err
		}
		return fmt.Errorf("linter found %d error(s); fix migration SQL or rerun with --force after manual review", lintErrors)
	}
	if lintErrors > 0 || lintWarnings > 0 {
		if err := renderLintResults(stdout, lintResults, lintErrors, lintWarnings); err != nil {
			return err
		}
	}

	if len(conflicts) > 0 && force {
		fmt.Fprintf(stdout, "%s continuing despite %d conflict(s) because --force was set\n", warn("WARN"), len(conflicts))
	}

	spinner := startTerminalSpinner(stdout, fmt.Sprintf("applying %d pending migration(s)", len(pending)))
	defer spinner.stop()

	if err := migration.RunUpWithEvents(ctx, pool, cfg, false, force, func(event migration.ApplyEvent) error {
		spinner.printLine("%s applied %s (%dms)\n", ok("OK"), event.Filename, event.ExecutionMs)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func pendingMigrations(applied []migration.MigrationRecord, files []migration.MigrationFile) []migration.MigrationFile {
	appliedByVersion := make(map[string]struct{}, len(applied))
	for _, record := range applied {
		if !record.RolledBack {
			appliedByVersion[record.Version] = struct{}{}
		}
	}

	pending := make([]migration.MigrationFile, 0)
	for _, file := range files {
		if _, ok := appliedByVersion[file.Version]; !ok {
			pending = append(pending, file)
		}
	}
	return pending
}
