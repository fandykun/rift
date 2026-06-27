package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fandykun/rift/internal/config"
	internaldiff "github.com/fandykun/rift/internal/diff"
	embedui "github.com/fandykun/rift/internal/embed"
	"github.com/fandykun/rift/internal/linter"
	"github.com/fandykun/rift/internal/migration"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewServer builds the Rift API router.
func NewServer(cfg *config.Config, pool *pgxpool.Pool) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(corsMiddleware)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	server := &server{cfg: cfg, pool: pool}
	router.Get("/healthz", server.handleHealth)
	router.Route("/api/v1", func(r chi.Router) {
		r.Use(server.authMiddleware)
		r.Get("/status", server.handleStatus)
		r.Get("/migrations", server.handleMigrations)
		r.Post("/migrations", server.handleCreateMigration)
		r.Get("/migrations/{version}", server.handleMigrationDetail)
		r.Put("/migrations/{version}", server.handleUpdateMigration)
		r.Get("/migrations/{version}/diff", server.handleMigrationDiff)
		r.Post("/migrate/up", server.handleMigrateUp)
		r.Post("/migrate/down", server.handleMigrateDown)
		r.Get("/history", server.handleHistory)
		r.Get("/lint", server.handleLint)
		r.Get("/team", server.handleTeam)
		r.Get("/conflicts", server.handleConflicts)
	})
	router.NotFound(serveSPA)
	return router
}

type server struct {
	cfg  *config.Config
	pool *pgxpool.Pool
}

var apiMigrationNamePattern = regexp.MustCompile(`[^a-z0-9]+`)

type statusResponse struct {
	Environment string     `json:"environment"`
	Counts      apiCounts  `json:"counts"`
	LastDeploy  *time.Time `json:"last_deploy,omitempty"`
}

type apiCounts struct {
	Applied    int `json:"applied"`
	Pending    int `json:"pending"`
	RolledBack int `json:"rolled_back"`
	Total      int `json:"total"`
}

type healthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment,omitempty"`
}

type migrationResponse struct {
	Version     string     `json:"version"`
	Filename    string     `json:"filename"`
	Status      string     `json:"status"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
	AppliedBy   string     `json:"applied_by,omitempty"`
	ExecutionMs int        `json:"execution_ms,omitempty"`
	HasLint     bool       `json:"has_lint"`
	UpSQL       string     `json:"up_sql,omitempty"`
	DownSQL     string     `json:"down_sql,omitempty"`
}

type createMigrationRequest struct {
	Name    string `json:"name"`
	UpSQL   string `json:"up_sql"`
	DownSQL string `json:"down_sql"`
}

type updateMigrationRequest struct {
	UpSQL   string `json:"up_sql"`
	DownSQL string `json:"down_sql"`
}

type lintResponse struct {
	ErrorCount   int             `json:"error_count"`
	WarningCount int             `json:"warning_count"`
	Results      []lintAPIResult `json:"results"`
}

type lintAPIResult struct {
	Filename string               `json:"filename"`
	Warnings []linter.LintWarning `json:"warnings"`
}

type migrateDownRequest struct {
	Steps int `json:"steps"`
}

type migrateDownResponse struct {
	Status string `json:"status"`
	Steps  int    `json:"steps"`
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{Status: "ok"}
	if s.cfg != nil {
		response.Environment = s.cfg.Environment
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	files, applied, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	counts := countMigrationState(files, applied)
	var lastDeploy *time.Time
	for _, record := range applied {
		if record.RolledBack {
			continue
		}
		appliedAt := record.AppliedAt
		if lastDeploy == nil || appliedAt.After(*lastDeploy) {
			lastDeploy = &appliedAt
		}
	}
	writeJSON(w, http.StatusOK, statusResponse{Environment: s.cfg.Environment, Counts: counts, LastDeploy: lastDeploy})
}

func (s *server) handleMigrations(w http.ResponseWriter, r *http.Request) {
	files, applied, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, migrationRows(files, applied, false))
}

func (s *server) handleCreateMigration(w http.ResponseWriter, r *http.Request) {
	if s.cfg == nil {
		writeError(w, http.StatusInternalServerError, "server config is not initialized")
		return
	}

	var request createMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decoding migration request: %v", err))
		return
	}

	file, err := createMigrationFiles(s.cfg.MigrationsDir, request, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	warnings := linter.LintSQL(file.UpSQL)
	writeJSON(w, http.StatusCreated, migrationResponse{
		Version:  file.Version,
		Filename: file.Filename,
		Status:   "pending",
		HasLint:  len(warnings) > 0,
		UpSQL:    file.UpSQL,
		DownSQL:  file.DownSQL,
	})
}

func (s *server) handleMigrationDetail(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	files, applied, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	for _, row := range migrationRows(files, applied, true) {
		if row.Version == version {
			writeJSON(w, http.StatusOK, row)
			return
		}
	}
	writeError(w, http.StatusNotFound, fmt.Sprintf("migration %q not found", version))
}

func (s *server) handleUpdateMigration(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	if s.cfg == nil {
		writeError(w, http.StatusInternalServerError, "server config is not initialized")
		return
	}

	var request updateMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decoding migration update request: %v", err))
		return
	}

	files, err := migration.LoadFiles(s.cfg.MigrationsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var target *migration.MigrationFile
	for i := range files {
		if files[i].Version == version {
			target = &files[i]
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("migration %q not found", version))
		return
	}

	if s.pool != nil {
		if err := migration.EnsureStateTable(r.Context(), s.pool); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		applied, err := migration.GetApplied(r.Context(), s.pool)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if migrationIsActivelyApplied(version, applied) {
			writeError(w, http.StatusConflict, fmt.Sprintf("migration %q has already been applied; create a new migration instead of editing applied SQL", version))
			return
		}
	}

	updated, err := updateMigrationFiles(s.cfg.MigrationsDir, *target, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	warnings := linter.LintSQL(updated.UpSQL)
	writeJSON(w, http.StatusOK, migrationResponse{
		Version:  updated.Version,
		Filename: updated.Filename,
		Status:   "pending",
		HasLint:  len(warnings) > 0,
		UpSQL:    updated.UpSQL,
		DownSQL:  updated.DownSQL,
	})
}

func (s *server) handleMigrationDiff(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	files, _, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	var target *migration.MigrationFile
	for i := range files {
		if files[i].Version == version {
			target = &files[i]
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("migration %q not found", version))
		return
	}
	live, err := internaldiff.IntrospectLive(r.Context(), s.pool, "public")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	expected, err := internaldiff.ApplyMigrationSQL(live, target.UpSQL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	content, err := internaldiff.RenderJSON(internaldiff.ComputeDiff(live, expected))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *server) handleHistory(w http.ResponseWriter, r *http.Request) {
	_, applied, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	sort.Slice(applied, func(i int, j int) bool { return applied[i].AppliedAt.After(applied[j].AppliedAt) })
	writeJSON(w, http.StatusOK, applied)
}

func (s *server) handleMigrateUp(w http.ResponseWriter, r *http.Request) {
	if s.cfg == nil {
		writeError(w, http.StatusInternalServerError, "server config is not initialized")
		return
	}
	if s.pool == nil {
		writeError(w, http.StatusInternalServerError, "database pool is not initialized")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	force := r.URL.Query().Get("force") == "true" || r.URL.Query().Get("force") == "1"
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	applied := 0
	err := migration.RunUpWithEvents(r.Context(), s.pool, s.cfg, false, force, func(event migration.ApplyEvent) error {
		applied++
		return writeSSE(w, flusher, "migration", map[string]interface{}{
			"status":       "applied",
			"version":      event.Version,
			"filename":     event.Filename,
			"execution_ms": event.ExecutionMs,
		})
	})
	if err != nil {
		_ = writeSSE(w, flusher, "error", map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	_ = writeSSE(w, flusher, "done", map[string]interface{}{"status": "done", "applied": applied})
}

func (s *server) handleMigrateDown(w http.ResponseWriter, r *http.Request) {
	if s.cfg == nil {
		writeError(w, http.StatusInternalServerError, "server config is not initialized")
		return
	}
	if s.pool == nil {
		writeError(w, http.StatusInternalServerError, "database pool is not initialized")
		return
	}
	var request migrateDownRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decoding rollback request: %v", err))
		return
	}
	if request.Steps <= 0 {
		writeError(w, http.StatusBadRequest, "steps must be greater than zero")
		return
	}
	if err := migration.RunDown(r.Context(), s.pool, s.cfg, request.Steps); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, migrateDownResponse{Status: "rolled-back", Steps: request.Steps})
}

func (s *server) handleLint(w http.ResponseWriter, r *http.Request) {
	files, applied, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	pending := pendingMigrationFiles(files, applied)
	results, errorCount, warningCount := lintFiles(pending)
	writeJSON(w, http.StatusOK, lintResponse{ErrorCount: errorCount, WarningCount: warningCount, Results: results})
}

func (s *server) handleTeam(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg.Team)
}

func (s *server) handleConflicts(w http.ResponseWriter, r *http.Request) {
	files, applied, ok := s.loadMigrationState(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, migration.DetectConflicts(applied, files))
}

func (s *server) loadMigrationState(w http.ResponseWriter, r *http.Request) ([]migration.MigrationFile, []migration.MigrationRecord, bool) {
	if s.cfg == nil {
		writeError(w, http.StatusInternalServerError, "server config is not initialized")
		return nil, nil, false
	}
	if s.pool == nil {
		writeError(w, http.StatusInternalServerError, "database pool is not initialized")
		return nil, nil, false
	}
	if err := migration.EnsureStateTable(r.Context(), s.pool); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, nil, false
	}
	files, err := migration.LoadFiles(s.cfg.MigrationsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, nil, false
	}
	applied, err := migration.GetApplied(r.Context(), s.pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, nil, false
	}
	return files, applied, true
}

func countMigrationState(files []migration.MigrationFile, applied []migration.MigrationRecord) apiCounts {
	pending := len(pendingMigrationFiles(files, applied))
	counts := apiCounts{Pending: pending, Total: len(files)}
	for _, record := range applied {
		if record.RolledBack {
			counts.RolledBack++
		} else {
			counts.Applied++
		}
	}
	return counts
}

func migrationRows(files []migration.MigrationFile, applied []migration.MigrationRecord, includeSQL bool) []migrationResponse {
	filesByVersion := make(map[string]migration.MigrationFile, len(files))
	for _, file := range files {
		filesByVersion[file.Version] = file
	}
	seen := make(map[string]struct{}, len(applied))
	rows := make([]migrationResponse, 0, len(files)+len(applied))
	for _, record := range applied {
		seen[record.Version] = struct{}{}
		status := "applied"
		if record.RolledBack {
			status = "rolled-back"
		}
		row := migrationResponse{Version: record.Version, Filename: record.Filename, Status: status, AppliedAt: &record.AppliedAt, AppliedBy: record.AppliedBy, ExecutionMs: record.ExecutionMs}
		if file, ok := filesByVersion[record.Version]; ok {
			warnings := linter.LintSQL(file.UpSQL)
			row.HasLint = len(warnings) > 0
			if includeSQL {
				row.UpSQL = file.UpSQL
				row.DownSQL = file.DownSQL
			}
		}
		rows = append(rows, row)
	}
	for _, file := range files {
		if _, ok := seen[file.Version]; ok {
			continue
		}
		warnings := linter.LintSQL(file.UpSQL)
		row := migrationResponse{Version: file.Version, Filename: file.Filename, Status: "pending", HasLint: len(warnings) > 0}
		if includeSQL {
			row.UpSQL = file.UpSQL
			row.DownSQL = file.DownSQL
		}
		rows = append(rows, row)
	}
	return rows
}

func pendingMigrationFiles(files []migration.MigrationFile, applied []migration.MigrationRecord) []migration.MigrationFile {
	appliedVersions := make(map[string]struct{}, len(applied))
	for _, record := range applied {
		if !record.RolledBack {
			appliedVersions[record.Version] = struct{}{}
		}
	}
	pending := make([]migration.MigrationFile, 0)
	for _, file := range files {
		if _, ok := appliedVersions[file.Version]; !ok {
			pending = append(pending, file)
		}
	}
	return pending
}

func lintFiles(files []migration.MigrationFile) ([]lintAPIResult, int, int) {
	results := make([]lintAPIResult, 0, len(files))
	errorCount := 0
	warningCount := 0
	for _, file := range files {
		warnings := linter.LintSQL(file.UpSQL)
		for _, warning := range warnings {
			if warning.Severity == "error" {
				errorCount++
			} else {
				warningCount++
			}
		}
		results = append(results, lintAPIResult{Filename: file.Filename, Warnings: warnings})
	}
	return results, errorCount, warningCount
}

func createMigrationFiles(migrationsDir string, request createMigrationRequest, createdAt time.Time) (migration.MigrationFile, error) {
	name := normalizeAPIMigrationName(request.Name)
	if name == "" {
		return migration.MigrationFile{}, fmt.Errorf("migration name must contain at least one letter or number")
	}
	if strings.TrimSpace(migrationsDir) == "" {
		return migration.MigrationFile{}, fmt.Errorf("migrations directory is not configured")
	}
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return migration.MigrationFile{}, fmt.Errorf("creating migrations directory %q: %w", migrationsDir, err)
	}

	version := createdAt.Format("20060102_150405")
	base := version + "_" + name
	upSQL := strings.TrimRight(request.UpSQL, "\n")
	if strings.TrimSpace(upSQL) == "" {
		upSQL = "-- Write your forward migration SQL here."
	}
	downSQL := strings.TrimRight(request.DownSQL, "\n")
	if strings.TrimSpace(downSQL) == "" {
		downSQL = "-- Write your rollback migration SQL here."
	}
	upSQL = migrationHeader(name, createdAt) + "\n" + upSQL + "\n"
	downSQL = migrationHeader(name, createdAt) + "\n" + downSQL + "\n"

	upFilename := base + ".up.sql"
	downFilename := base + ".down.sql"
	if err := writeNewMigrationFile(filepath.Join(migrationsDir, upFilename), upSQL); err != nil {
		return migration.MigrationFile{}, err
	}
	if err := writeNewMigrationFile(filepath.Join(migrationsDir, downFilename), downSQL); err != nil {
		_ = os.Remove(filepath.Join(migrationsDir, upFilename))
		return migration.MigrationFile{}, err
	}

	return migration.MigrationFile{
		Version:  version,
		Filename: base,
		UpSQL:    upSQL,
		DownSQL:  downSQL,
		Checksum: migration.Checksum([]byte(upSQL)),
	}, nil
}

func updateMigrationFiles(migrationsDir string, file migration.MigrationFile, request updateMigrationRequest) (migration.MigrationFile, error) {
	upSQL := strings.TrimRight(request.UpSQL, "\n") + "\n"
	downSQL := strings.TrimRight(request.DownSQL, "\n") + "\n"
	if strings.TrimSpace(upSQL) == "" || strings.TrimSpace(downSQL) == "" {
		return migration.MigrationFile{}, fmt.Errorf("up_sql and down_sql are required")
	}

	upPath := filepath.Join(migrationsDir, file.Filename+".up.sql")
	downPath := filepath.Join(migrationsDir, file.Filename+".down.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0o644); err != nil {
		return migration.MigrationFile{}, fmt.Errorf("writing up migration %q: %w", upPath, err)
	}
	if err := os.WriteFile(downPath, []byte(downSQL), 0o644); err != nil {
		return migration.MigrationFile{}, fmt.Errorf("writing down migration %q: %w", downPath, err)
	}

	file.UpSQL = upSQL
	file.DownSQL = downSQL
	file.Checksum = migration.Checksum([]byte(upSQL))
	return file, nil
}

func migrationIsActivelyApplied(version string, applied []migration.MigrationRecord) bool {
	for _, record := range applied {
		if record.Version == version && !record.RolledBack {
			return true
		}
	}
	return false
}

func normalizeAPIMigrationName(rawName string) string {
	name := strings.ToLower(strings.TrimSpace(rawName))
	name = apiMigrationNamePattern.ReplaceAllString(name, "_")
	return strings.Trim(name, "_")
}

func migrationHeader(name string, createdAt time.Time) string {
	return fmt.Sprintf("-- Migration: %s | Created: %s", name, createdAt.Format(time.RFC3339))
}

func writeNewMigrationFile(path string, content string) error {
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

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg == nil || s.cfg.Server.Token == "" {
			next.ServeHTTP(w, r)
			return
		}
		expected := "Bearer " + s.cfg.Server.Token
		if r.Header.Get("Authorization") != expected {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(message)})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, value interface{}) error {
	content, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encoding SSE event %q: %w", event, err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, content); err != nil {
		return fmt.Errorf("writing SSE event %q: %w", event, err)
	}
	flusher.Flush()
	return nil
}

func serveSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "API route not found")
		return
	}
	uiFS, err := fs.Sub(embedui.FS, "ui")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("loading embedded UI: %v", err))
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" {
		if file, err := uiFS.Open(path); err == nil {
			_ = file.Close()
			http.FileServer(http.FS(uiFS)).ServeHTTP(w, r)
			return
		}
	}
	content, err := fs.ReadFile(uiFS, "index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("reading embedded UI index: %v", err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
