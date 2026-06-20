package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fandykun/rift/internal/api"
	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewServerCommand returns the `rift server` command for the API and embedded dashboard.
func NewServerCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the Rift API server and embedded dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServer(cmd.Context(), cmd.OutOrStdout(), *configPath)
		},
	}
}

// RunServer starts the Rift HTTP server and blocks until it exits.
func RunServer(ctx context.Context, stdout io.Writer, configPath string) error {
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

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(cfg, pool),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ok := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(stdout, "%s Rift server listening on http://localhost:%d\n", ok("OK"), cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("starting Rift server on %s: %w", addr, err)
	}
	return nil
}
