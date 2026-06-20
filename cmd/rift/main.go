package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fandykun/rift/internal/cli"
	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

func main() {
	if err := newRootCommand().Execute(); err != nil {
		if errors.Is(err, cli.ErrSchemaDiffDetected) || errors.Is(err, cli.ErrLintErrorsFound) {
			os.Exit(1)
		}
		color.New(color.FgRed).Fprintln(os.Stderr, "FAIL", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var configPath string
	var verbose bool

	rootCmd := &cobra.Command{
		Use:           "rift",
		Short:         "Self-hosted PostgreSQL migration manager",
		Long:          "Rift is a CLI and web dashboard for authoring, previewing, applying, and auditing PostgreSQL migrations.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "./rift.yaml", "path to Rift config file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")

	rootCmd.AddCommand(
		cli.NewMigrationCommand(&configPath),
		cli.NewUpCommand(&configPath),
		cli.NewDownCommand(&configPath),
		cli.NewStatusCommand(&configPath),
		cli.NewDiffCommand(&configPath),
		cli.NewServerCommand(&configPath),
		cli.NewLintCommand(&configPath),
		newConfigCommand(&configPath),
	)

	return rootCmd
}

func placeholderCommand(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%s command is not implemented yet", cmd.Name())
		},
	}
}

func newConfigCommand(configPath *string) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate Rift configuration",
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Validate configuration and database connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigCheck(cmd.Context(), *configPath)
		},
	})

	return configCmd
}

func runConfigCheck(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	ok := color.New(color.FgGreen).SprintFunc()
	muted := color.New(color.FgHiBlack).SprintFunc()

	fmt.Printf("%s config loaded %s\n", ok("OK"), muted(configPath))
	fmt.Printf("%s environment %s\n", ok("OK"), cfg.Environment)
	fmt.Printf("%s migrations dir %s\n", ok("OK"), cfg.MigrationsDir)

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := db.NewPool(checkCtx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Ping(checkCtx, pool); err != nil {
		return err
	}

	fmt.Printf("%s database connection verified\n", ok("OK"))
	return nil
}
