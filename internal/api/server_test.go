package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fandykun/rift/internal/config"
	"github.com/fandykun/rift/internal/db"
	"github.com/fandykun/rift/internal/migration"
)

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	router := NewServer(&config.Config{Server: config.ServerConfig{Token: "secret"}}, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestTeamEndpoint(t *testing.T) {
	router := NewServer(&config.Config{
		Server: config.ServerConfig{Token: "secret"},
		Team:   []config.TeamMember{{Name: "Fandy", Email: "fandy@example.com", Role: "admin"}},
	}, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/team", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "fandy@example.com") {
		t.Fatalf("expected team JSON, got %s", response.Body.String())
	}
}

func TestStatusAndMigrationsIntegration(t *testing.T) {
	cfg := setupAPIIntegration(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	defer pool.Close()
	router := NewServer(cfg, pool)

	statusRecorder := performAPIRequest(t, router, http.MethodGet, "/api/v1/status", cfg.Server.Token, nil)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status endpoint = %d, body = %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var status statusResponse
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &status); err != nil {
		t.Fatalf("decoding status response: %v", err)
	}
	if status.Environment != "test" || status.Counts.Applied != 2 || status.Counts.Pending != 1 || status.Counts.RolledBack != 1 {
		t.Fatalf("unexpected status response: %+v", status)
	}

	migrationsResponse := performAPIRequest(t, router, http.MethodGet, "/api/v1/migrations", cfg.Server.Token, nil)
	if migrationsResponse.Code != http.StatusOK {
		t.Fatalf("migrations endpoint = %d, body = %s", migrationsResponse.Code, migrationsResponse.Body.String())
	}
	var migrations []migrationResponse
	if err := json.Unmarshal(migrationsResponse.Body.Bytes(), &migrations); err != nil {
		t.Fatalf("decoding migrations response: %v", err)
	}
	if len(migrations) != 4 {
		t.Fatalf("expected 4 migration rows, got %+v", migrations)
	}
	if !containsMigrationStatus(migrations, "20260620_170000", "applied") || !containsMigrationStatus(migrations, "20260620_170001", "pending") || !containsMigrationStatus(migrations, "20260620_170002", "rolled-back") {
		t.Fatalf("unexpected migration rows: %+v", migrations)
	}
}

func TestLintAndConflictsIntegration(t *testing.T) {
	cfg := setupAPIIntegration(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	defer pool.Close()
	router := NewServer(cfg, pool)

	lintResponseRecorder := performAPIRequest(t, router, http.MethodGet, "/api/v1/lint", cfg.Server.Token, nil)
	if lintResponseRecorder.Code != http.StatusOK {
		t.Fatalf("lint endpoint = %d, body = %s", lintResponseRecorder.Code, lintResponseRecorder.Body.String())
	}
	var lintResult lintResponse
	if err := json.Unmarshal(lintResponseRecorder.Body.Bytes(), &lintResult); err != nil {
		t.Fatalf("decoding lint response: %v", err)
	}
	if lintResult.WarningCount != 1 || lintResult.ErrorCount != 0 {
		t.Fatalf("unexpected lint result: %+v", lintResult)
	}

	conflictsResponse := performAPIRequest(t, router, http.MethodGet, "/api/v1/conflicts", cfg.Server.Token, nil)
	if conflictsResponse.Code != http.StatusOK {
		t.Fatalf("conflicts endpoint = %d, body = %s", conflictsResponse.Code, conflictsResponse.Body.String())
	}
	if !strings.Contains(conflictsResponse.Body.String(), "MISSING_FILE") {
		t.Fatalf("expected missing-file conflict, got %s", conflictsResponse.Body.String())
	}
}

func TestMigrateUpSSEAndDownIntegration(t *testing.T) {
	cfg := setupAPIMutationIntegration(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("creating pool: %v", err)
	}
	defer pool.Close()
	router := NewServer(cfg, pool)

	upResponse := performAPIRequest(t, router, http.MethodPost, "/api/v1/migrate/up", cfg.Server.Token, nil)
	if upResponse.Code != http.StatusOK {
		t.Fatalf("migrate up status = %d, body = %s", upResponse.Code, upResponse.Body.String())
	}
	body := upResponse.Body.String()
	if !strings.Contains(body, "event: migration") || !strings.Contains(body, "event: done") || !strings.Contains(body, "20260620_180000_create_api_widgets") {
		t.Fatalf("expected migration and done SSE events, got %q", body)
	}

	downResponse := performAPIRequest(t, router, http.MethodPost, "/api/v1/migrate/down", cfg.Server.Token, strings.NewReader(`{"steps":1}`))
	if downResponse.Code != http.StatusOK {
		t.Fatalf("migrate down status = %d, body = %s", downResponse.Code, downResponse.Body.String())
	}
	if !strings.Contains(downResponse.Body.String(), "rolled-back") {
		t.Fatalf("expected rollback response, got %q", downResponse.Body.String())
	}
}

func setupAPIMutationIntegration(t *testing.T) *config.Config {
	t.Helper()
	databaseURL := os.Getenv("RIFT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RIFT_TEST_DATABASE_URL is not set")
	}
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating setup pool: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS api_widgets CASCADE; DROP TABLE IF EXISTS _rift_migrations CASCADE;`); err != nil {
		t.Fatalf("resetting database: %v", err)
	}

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	writeFile(t, migrationsDir, "20260620_180000_create_api_widgets.up.sql", "CREATE TABLE api_widgets (id BIGSERIAL PRIMARY KEY);")
	writeFile(t, migrationsDir, "20260620_180000_create_api_widgets.down.sql", "DROP TABLE api_widgets;")
	return &config.Config{Environment: "test", DatabaseURL: databaseURL, MigrationsDir: migrationsDir, Author: "api-test", Server: config.ServerConfig{Token: "secret"}}
}

func setupAPIIntegration(t *testing.T) *config.Config {
	t.Helper()
	databaseURL := os.Getenv("RIFT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RIFT_TEST_DATABASE_URL is not set")
	}
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	pool, err := db.NewPool(ctx, &config.Config{DatabaseURL: databaseURL})
	if err != nil {
		t.Fatalf("creating setup pool: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS api_posts CASCADE; DROP TABLE IF EXISTS api_users CASCADE; DROP TABLE IF EXISTS _rift_migrations CASCADE;`); err != nil {
		t.Fatalf("resetting database: %v", err)
	}
	if err := migration.EnsureStateTable(ctx, pool); err != nil {
		t.Fatalf("ensuring state table: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning setup transaction: %v", err)
	}
	if err := migration.RecordApplied(ctx, tx, migration.MigrationRecord{Version: "20260620_170000", Filename: "20260620_170000_create_api_users", Checksum: migration.Checksum([]byte("CREATE TABLE api_users (id BIGSERIAL PRIMARY KEY);")), AppliedBy: "api-test", ExecutionMs: 12}); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording applied migration: %v", err)
	}
	if err := migration.RecordApplied(ctx, tx, migration.MigrationRecord{Version: "20260620_170002", Filename: "20260620_170002_missing_file", Checksum: "missing-checksum", AppliedBy: "api-test", ExecutionMs: 7}); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording missing-file migration: %v", err)
	}
	if err := migration.RecordRolledBack(ctx, tx, "20260620_170002"); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording rollback: %v", err)
	}
	if err := migration.RecordApplied(ctx, tx, migration.MigrationRecord{Version: "20260620_165959", Filename: "20260620_165959_missing_active", Checksum: "missing-active", AppliedBy: "api-test", ExecutionMs: 3}); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("recording active missing-file migration: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("committing setup transaction: %v", err)
	}

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	writeFile(t, migrationsDir, "20260620_170000_create_api_users.up.sql", "CREATE TABLE api_users (id BIGSERIAL PRIMARY KEY);")
	writeFile(t, migrationsDir, "20260620_170000_create_api_users.down.sql", "DROP TABLE api_users;")
	writeFile(t, migrationsDir, "20260620_170001_create_api_posts.up.sql", "CREATE INDEX api_posts_user_id_idx ON api_posts (user_id);")
	writeFile(t, migrationsDir, "20260620_170001_create_api_posts.down.sql", "DROP INDEX IF EXISTS api_posts_user_id_idx;")
	return &config.Config{Environment: "test", DatabaseURL: databaseURL, MigrationsDir: migrationsDir, Server: config.ServerConfig{Token: "secret"}, Team: []config.TeamMember{{Name: "Fandy", Email: "fandy@example.com", Role: "admin"}}}
}

func performAPIRequest(t *testing.T, router http.Handler, method string, path string, token string, body *strings.Reader) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody strings.Reader
	if body != nil {
		requestBody = *body
	}
	request := httptest.NewRequest(method, path, &requestBody)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func containsMigrationStatus(rows []migrationResponse, version string, status string) bool {
	for _, row := range rows {
		if row.Version == version && row.Status == status {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, dir string, name string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

var _ = time.Second
