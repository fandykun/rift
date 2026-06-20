package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/linter"
	"github.com/fandykun/rift/internal/migration"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ErrLintErrorsFound is returned when lint errors are found and warn-only mode is disabled.
var ErrLintErrorsFound = errors.New("lint errors found")

// NewLintCommand returns the `rift lint` command for checking migration SQL.
func NewLintCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "lint [file]",
		Short: "Lint migration SQL for dangerous DDL patterns",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := ""
			if len(args) == 1 {
				filePath = args[0]
			}
			return RunLint(cmd.Context(), cmd.OutOrStdout(), *configPath, filePath)
		},
	}
}

// RunLint lints either one SQL file or all pending migrations from config.
func RunLint(ctx context.Context, stdout io.Writer, configPath string, filePath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	results, err := lintTargets(ctx, cfg, filePath)
	if err != nil {
		return err
	}

	errorCount := 0
	warningCount := 0
	for _, result := range results {
		for _, warning := range result.Warnings {
			if warning.Severity == "error" {
				errorCount++
			} else {
				warningCount++
			}
		}
	}

	if err := renderLintResults(stdout, results, errorCount, warningCount); err != nil {
		return err
	}
	if errorCount > 0 && !cfg.Linter.WarnOnly {
		return ErrLintErrorsFound
	}
	return nil
}

type lintResult struct {
	Name     string
	Warnings []linter.LintWarning
}

func lintTargets(ctx context.Context, cfg *config.Config, filePath string) ([]lintResult, error) {
	if filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading lint target %q: %w", filePath, err)
		}
		return []lintResult{{Name: filePath, Warnings: linter.LintSQL(string(content))}}, nil
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(checkCtx, cfg)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	if err := db.Ping(checkCtx, pool); err != nil {
		return nil, err
	}
	if err := migration.EnsureStateTable(checkCtx, pool); err != nil {
		return nil, err
	}
	files, err := migration.LoadFiles(cfg.MigrationsDir)
	if err != nil {
		return nil, err
	}
	applied, err := migration.GetApplied(checkCtx, pool)
	if err != nil {
		return nil, err
	}
	pending := pendingMigrations(applied, files)
	results := make([]lintResult, 0, len(pending))
	for _, file := range pending {
		results = append(results, lintResult{Name: file.Filename, Warnings: linter.LintSQL(file.UpSQL)})
	}
	return results, nil
}

func renderLintResults(stdout io.Writer, results []lintResult, errorCount int, warningCount int) error {
	ok := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	muted := color.New(color.FgHiBlack).SprintFunc()

	if errorCount == 0 && warningCount == 0 {
		_, err := fmt.Fprintf(stdout, "%s no lint warnings\n", ok("OK"))
		return err
	}

	fmt.Fprintf(stdout, "%s %d error(s), %d warning(s)\n", yellow("LINT"), errorCount, warningCount)
	writer := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	for _, result := range results {
		for _, warning := range result.Warnings {
			severity := yellow(warning.Severity)
			if warning.Severity == "error" {
				severity = red(warning.Severity)
			}
			fmt.Fprintf(writer, "%s:%d\t%s\t%s\t%s\n", result.Name, warning.Line, severity, warning.Pattern, warning.Message)
			fmt.Fprintf(writer, "\t%s\t%s\t%s\n", muted("fix"), "", warning.Suggestion)
		}
	}
	return writer.Flush()
}
