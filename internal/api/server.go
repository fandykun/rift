package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
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
		r.Get("/migrations/{version}", server.handleMigrationDetail)
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
