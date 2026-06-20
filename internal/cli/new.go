package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var migrationNamePattern = regexp.MustCompile(`[^a-z0-9]+`)

// NewMigrationCommand returns the `rift new` command for creating migration file pairs.
func NewMigrationCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new timestamped migration pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunNew(cmd.OutOrStdout(), *configPath, args[0], time.Now())
		},
	}
}

// RunNew creates timestamped up/down SQL migration files in the configured migrations directory.
func RunNew(stdout io.Writer, configPath string, rawName string, createdAt time.Time) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	name := normalizeMigrationName(rawName)
	if name == "" {
		return fmt.Errorf("migration name %q must contain at least one letter or number", rawName)
	}

	if err := os.MkdirAll(cfg.MigrationsDir, 0o755); err != nil {
		return fmt.Errorf("creating migrations directory %q: %w", cfg.MigrationsDir, err)
	}

	version := createdAt.Format("20060102_150405")
	base := version + "_" + name
	upPath := filepath.Join(cfg.MigrationsDir, base+".up.sql")
	downPath := filepath.Join(cfg.MigrationsDir, base+".down.sql")

	upContent := migrationHeader(name, createdAt) + "\n-- Write your forward migration SQL here.\n"
	downContent := migrationHeader(name, createdAt) + "\n-- Write your rollback migration SQL here.\n"

	if err := writeNewFile(upPath, upContent); err != nil {
		return err
	}
	if err := writeNewFile(downPath, downContent); err != nil {
		return err
	}

	ok := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(stdout, "%s created %s\n", ok("OK"), upPath)
	fmt.Fprintf(stdout, "%s created %s\n", ok("OK"), downPath)
	return nil
}

func normalizeMigrationName(rawName string) string {
	name := strings.ToLower(strings.TrimSpace(rawName))
	name = migrationNamePattern.ReplaceAllString(name, "_")
	return strings.Trim(name, "_")
}

func migrationHeader(name string, createdAt time.Time) string {
	return fmt.Sprintf("-- Migration: %s | Created: %s", name, createdAt.Format(time.RFC3339))
}

func writeNewFile(path string, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("migration file already exists: %s", path)
		}
		return fmt.Errorf("creating migration file %q: %w", path, err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("writing migration file %q: %w", path, err)
	}
	return nil
}
