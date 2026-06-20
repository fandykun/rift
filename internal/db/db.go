package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fandykun/rift/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a PostgreSQL connection pool from Rift config.
func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg == nil {
		return nil, errors.New("config is required to create database pool")
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, errors.New("database_url is required; set it in rift.yaml or RIFT_DATABASE_URL")
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("creating PostgreSQL pool: %w", err)
	}

	return pool, nil
}

// Ping verifies that the PostgreSQL pool can acquire a live connection.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("database pool is nil")
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("pinging PostgreSQL database: %w", err)
	}
	return nil
}
