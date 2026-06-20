package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	DefaultEnvironment   = "development"
	DefaultMigrationsDir = "./migrations"
	DefaultPort          = 7878
)

// Config is Rift's runtime configuration loaded from defaults, YAML, and environment variables.
type Config struct {
	Environment   string       `yaml:"environment"`
	DatabaseURL   string       `yaml:"database_url"`
	MigrationsDir string       `yaml:"migrations_dir"`
	Author        string       `yaml:"author"`
	Server        ServerConfig `yaml:"server"`
	Linter        LinterConfig `yaml:"linter"`
	Team          []TeamMember `yaml:"team"`
}

// ServerConfig controls the embedded HTTP API/dashboard server.
type ServerConfig struct {
	Port  int    `yaml:"port"`
	Token string `yaml:"token"`
}

// LinterConfig controls dangerous-DDL linter behavior.
type LinterConfig struct {
	WarnOnly bool `yaml:"warn_only"`
}

// TeamMember describes one collaborator displayed in the dashboard.
type TeamMember struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email" json:"email"`
	Role  string `yaml:"role" json:"role"`
}

// Default returns a Config populated with safe local defaults.
func Default() Config {
	return Config{
		Environment:   DefaultEnvironment,
		MigrationsDir: DefaultMigrationsDir,
		Server: ServerConfig{
			Port: DefaultPort,
		},
	}
}

// Load reads config from path, then applies RIFT_* environment overrides.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("reading config file %q: %w", path, err)
			}
		} else if len(content) > 0 {
			if err := yaml.Unmarshal(content, &cfg); err != nil {
				return nil, fmt.Errorf("parsing config file %q: %w", path, err)
			}
		}
	}

	if value := os.Getenv("RIFT_DATABASE_URL"); value != "" {
		cfg.DatabaseURL = value
	}
	if value := os.Getenv("RIFT_ENV"); value != "" {
		cfg.Environment = value
	}
	if value := os.Getenv("RIFT_MIGRATIONS_DIR"); value != "" {
		cfg.MigrationsDir = value
	}
	if value := os.Getenv("RIFT_PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("RIFT_PORT must be a valid TCP port, got %q", value)
		}
		cfg.Server.Port = port
	}
	if value := os.Getenv("RIFT_TOKEN"); value != "" {
		cfg.Server.Token = value
	}

	return &cfg, nil
}
