package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewStatusCommand returns the `rift status` command for displaying migration state.
func NewStatusCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show applied and pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunStatus(cmd.Context(), cmd.OutOrStdout(), *configPath)
		},
	}
}

// RunStatus renders applied, pending, and rolled-back migrations as a table.
func RunStatus(ctx context.Context, stdout io.Writer, configPath string) error {
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

	rows := statusRows(applied, files)
	writer := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "VERSION\tFILENAME\tSTATUS\tAPPLIED_AT\tAPPLIED_BY\tTIME_MS")
	for _, row := range rows {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", row.Version, row.Filename, colorStatus(row.Status), row.AppliedAt, row.AppliedBy, row.ExecutionMs)
	}
	return writer.Flush()
}

type statusRow struct {
	Version     string
	Filename    string
	Status      string
	AppliedAt   string
	AppliedBy   string
	ExecutionMs string
}

func statusRows(applied []migration.MigrationRecord, files []migration.MigrationFile) []statusRow {
	rows := make([]statusRow, 0, len(applied)+len(files))
	seenVersions := make(map[string]struct{}, len(applied))

	for _, record := range applied {
		seenVersions[record.Version] = struct{}{}
		status := "applied"
		if record.RolledBack {
			status = "rolled-back"
		}
		rows = append(rows, statusRow{
			Version:     record.Version,
			Filename:    record.Filename,
			Status:      status,
			AppliedAt:   record.AppliedAt.Format(time.RFC3339),
			AppliedBy:   record.AppliedBy,
			ExecutionMs: fmt.Sprintf("%d", record.ExecutionMs),
		})
	}

	for _, file := range files {
		if _, ok := seenVersions[file.Version]; ok {
			continue
		}
		rows = append(rows, statusRow{
			Version:     file.Version,
			Filename:    file.Filename,
			Status:      "pending",
			AppliedAt:   "-",
			AppliedBy:   "-",
			ExecutionMs: "-",
		})
	}

	return rows
}

func colorStatus(status string) string {
	switch status {
	case "applied":
		return color.New(color.FgGreen).Sprint(status)
	case "pending":
		return color.New(color.FgYellow).Sprint(status)
	case "rolled-back":
		return color.New(color.FgHiBlack).Sprint(status)
	default:
		return status
	}
}
