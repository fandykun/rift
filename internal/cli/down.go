package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewDownCommand returns the `rift down` command for rolling back applied migrations.
func NewDownCommand(configPath *string) *cobra.Command {
	var steps int

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Roll back applied migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDown(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), *configPath, steps)
		},
	}
	cmd.Flags().IntVar(&steps, "steps", 1, "number of applied migrations to roll back")
	return cmd
}

// RunDown confirms and rolls back applied migrations.
func RunDown(ctx context.Context, stdin io.Reader, stdout io.Writer, configPath string, steps int) error {
	if steps <= 0 {
		return fmt.Errorf("steps must be greater than zero")
	}

	migrationWord := "migration"
	if steps != 1 {
		migrationWord = "migrations"
	}
	fmt.Fprintf(stdout, "Roll back %d %s? (yes/no): ", steps, migrationWord)
	answer, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading rollback confirmation: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(answer)) != "yes" {
		fmt.Fprintln(stdout, "Rollback aborted")
		return nil
	}

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
	if err := migration.RunDown(checkCtx, pool, cfg, steps); err != nil {
		return err
	}

	ok := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(stdout, "%s rolled back %d %s\n", ok("OK"), steps, migrationWord)
	return nil
}
