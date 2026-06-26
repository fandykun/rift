package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fatih/color"
)

type doctorSeverity string

const (
	doctorOK   doctorSeverity = "OK"
	doctorWarn doctorSeverity = "WARN"
	doctorFail doctorSeverity = "FAIL"
)

type doctorCheck struct {
	Severity doctorSeverity
	Name     string
	Message  string
}

func runConfigDoctor(ctx context.Context, stdout io.Writer, configPath string, skipDB bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	checks := evaluateConfigDoctor(ctx, cfg, skipDB)
	writeDoctorReport(stdout, cfg, checks)

	failures := 0
	for _, check := range checks {
		if check.Severity == doctorFail {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("config doctor found %d blocking issue(s)", failures)
	}
	return nil
}

func evaluateConfigDoctor(ctx context.Context, cfg *config.Config, skipDB bool) []doctorCheck {
	checks := []doctorCheck{
		checkEnvironment(cfg.Environment),
		checkPort(cfg.Server.Port),
		checkDatabaseURL(cfg.DatabaseURL),
		checkToken(cfg.Environment, cfg.Server.Token),
		checkMigrationsDir(cfg.MigrationsDir),
	}

	if cfg.DatabaseURL == "" {
		checks = append(checks, doctorCheck{Severity: doctorWarn, Name: "database connectivity", Message: "skipped because RIFT_DATABASE_URL is not set"})
	} else if skipDB {
		checks = append(checks, doctorCheck{Severity: doctorWarn, Name: "database connectivity", Message: "skipped because --skip-db was set"})
	} else {
		checks = append(checks, checkDatabaseConnectivity(ctx, cfg))
	}

	return checks
}

func checkEnvironment(environment string) doctorCheck {
	if strings.TrimSpace(environment) == "" {
		return doctorCheck{Severity: doctorFail, Name: "environment", Message: "environment is empty; set RIFT_ENV or environment in rift.yaml"}
	}
	return doctorCheck{Severity: doctorOK, Name: "environment", Message: fmt.Sprintf("using %q", environment)}
}

func checkPort(port int) doctorCheck {
	if port <= 0 || port > 65535 {
		return doctorCheck{Severity: doctorFail, Name: "server port", Message: fmt.Sprintf("%d is not a valid TCP port", port)}
	}
	return doctorCheck{Severity: doctorOK, Name: "server port", Message: fmt.Sprintf("listening on %d", port)}
}

func checkDatabaseURL(databaseURL string) doctorCheck {
	if databaseURL == "" {
		return doctorCheck{Severity: doctorFail, Name: "database url", Message: "RIFT_DATABASE_URL or database_url is required"}
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return doctorCheck{Severity: doctorFail, Name: "database url", Message: "database URL must be a valid PostgreSQL connection URL"}
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return doctorCheck{Severity: doctorFail, Name: "database url", Message: fmt.Sprintf("unsupported scheme %q; use postgres:// or postgresql://", parsed.Scheme)}
	}
	return doctorCheck{Severity: doctorOK, Name: "database url", Message: fmt.Sprintf("configured for %s", redactDatabaseURL(databaseURL))}
}

func checkToken(environment string, token string) doctorCheck {
	if token == "" {
		return doctorCheck{Severity: doctorFail, Name: "api token", Message: "RIFT_TOKEN or server.token is required"}
	}
	if token == "local-dev-token" && environment != "development" {
		return doctorCheck{Severity: doctorFail, Name: "api token", Message: "local-dev-token must not be used outside development"}
	}
	if len(token) < 32 && environment != "development" {
		return doctorCheck{Severity: doctorFail, Name: "api token", Message: "token is too short for public deployment; generate at least 32 random characters"}
	}
	if len(token) < 16 {
		return doctorCheck{Severity: doctorWarn, Name: "api token", Message: "token is short; use a generated secret before sharing this instance"}
	}
	return doctorCheck{Severity: doctorOK, Name: "api token", Message: "configured without exposing the secret"}
}

func checkMigrationsDir(migrationsDir string) doctorCheck {
	if strings.TrimSpace(migrationsDir) == "" {
		return doctorCheck{Severity: doctorFail, Name: "migrations dir", Message: "RIFT_MIGRATIONS_DIR or migrations_dir is required"}
	}
	info, err := os.Stat(migrationsDir)
	if err != nil {
		return doctorCheck{Severity: doctorFail, Name: "migrations dir", Message: fmt.Sprintf("%s is not accessible: %v", migrationsDir, err)}
	}
	if !info.IsDir() {
		return doctorCheck{Severity: doctorFail, Name: "migrations dir", Message: fmt.Sprintf("%s is not a directory", migrationsDir)}
	}
	return doctorCheck{Severity: doctorOK, Name: "migrations dir", Message: fmt.Sprintf("%s exists", migrationsDir)}
}

func checkDatabaseConnectivity(ctx context.Context, cfg *config.Config) doctorCheck {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := db.NewPool(checkCtx, cfg)
	if err != nil {
		return doctorCheck{Severity: doctorFail, Name: "database connectivity", Message: fmt.Sprintf("could not create database pool: %v", err)}
	}
	defer pool.Close()

	if err := db.Ping(checkCtx, pool); err != nil {
		return doctorCheck{Severity: doctorFail, Name: "database connectivity", Message: fmt.Sprintf("database ping failed: %v", err)}
	}
	return doctorCheck{Severity: doctorOK, Name: "database connectivity", Message: "database ping succeeded"}
}

func writeDoctorReport(stdout io.Writer, cfg *config.Config, checks []doctorCheck) {
	ok := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()
	fail := color.New(color.FgRed).SprintFunc()

	fmt.Fprintf(stdout, "Rift config doctor for %s\n", cfg.Environment)
	for _, check := range checks {
		label := string(check.Severity)
		switch check.Severity {
		case doctorOK:
			label = ok(label)
		case doctorWarn:
			label = warn(label)
		case doctorFail:
			label = fail(label)
		}
		fmt.Fprintf(stdout, "%s %-22s %s\n", label, check.Name, check.Message)
	}
}

func redactDatabaseURL(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "[redacted-invalid-url]"
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username == "" {
			parsed.User = url.UserPassword("[redacted]", "[redacted]")
		} else {
			parsed.User = url.UserPassword(username, "[redacted]")
		}
	}
	return parsed.String()
}
