package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var configPath string
	var verbose bool

	rootCmd := &cobra.Command{
		Use:     "rift",
		Short:   "Self-hosted PostgreSQL migration manager",
		Long:    "Rift is a CLI and web dashboard for authoring, previewing, applying, and auditing PostgreSQL migrations.",
		Version: version,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "./rift.yaml", "path to Rift config file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")

	rootCmd.AddCommand(
		placeholderCommand("new", "Create a new timestamped migration pair"),
		placeholderCommand("up", "Apply pending migrations"),
		placeholderCommand("down", "Roll back applied migrations"),
		placeholderCommand("status", "Show applied and pending migrations"),
		placeholderCommand("diff", "Compare pending migrations against the live database schema"),
		placeholderCommand("server", "Start the Rift API server and embedded dashboard"),
		placeholderCommand("lint", "Lint migration SQL for dangerous DDL patterns"),
		newConfigCommand(),
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

func newConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate Rift configuration",
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Validate configuration and database connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("config check command is not implemented yet")
		},
	})

	return configCmd
}
