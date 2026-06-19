# BUILD_PHASES.md — Rift

Each phase is a self-contained, committable, testable unit. The agent completes one phase fully before moving to the next. Every phase ends with `git add -A && git commit -m "phase(N): <description>" && git push`.

No phase is skipped. No phase is merged unless its verification checklist passes.

---

## Phase 0 — Project Scaffold & Tooling

**Goal:** Establish the monorepo structure, Go module, React app, Makefile, and CI config. No business logic yet.

**Tasks:**
1. Initialize Go module: `go mod init github.com/yourhandle/rift`
2. Create directory tree:
   ```
   cmd/rift/main.go
   internal/cli/
   internal/migration/
   internal/diff/
   internal/linter/
   internal/api/
   internal/config/
   internal/embed/ui/.gitkeep
   web/
   migrations/
   docker/
   ```
3. Scaffold `web/` with Vite + React + TypeScript: `npm create vite@latest web -- --template react-ts`
4. Install frontend deps: `@tanstack/react-query`, `zustand`, `tailwindcss`, `@tailwindcss/forms`, `@tailwindcss/container-queries`, `react-router-dom`, `@uiw/react-codemirror`, `@codemirror/lang-sql`, `@codemirror/language`, `vitest`, `@testing-library/react`
5. Configure Tailwind in `web/` with the exact colors, typography, spacing, border radius, and plugin config from `DESIGN.md` Section 11
6. Write `Makefile` with targets:
   - `make build-web` → `cd web && npm run build && cp -r dist/* ../internal/embed/ui/`
   - `make build` → `make build-web && go build -o rift ./cmd/rift`
   - `make dev-web` → `cd web && npm run dev`
   - `make dev-api` → `go run ./cmd/rift server`
   - `make test` → `go test ./... && cd web && npm run test`
7. Write `rift.yaml.example`
8. Write `.gitignore` (Go + Node + build artifacts)
9. Write root `README.md` with project description and badge placeholders
10. Add `docker/Dockerfile` (multi-stage: Node build → Go build → minimal runtime)
11. Add `docker-compose.yml` with `rift` service + `postgres` service
12. Load Google Fonts for Geist, Inter, JetBrains Mono, and Material Symbols Outlined in the frontend entry HTML

**Verification:**
- [ ] `go mod tidy` runs without error
- [ ] `cd web && npm install && npm run build` succeeds
- [ ] `make build` produces a `./rift` binary
- [ ] `./rift --help` prints usage without panicking
- [ ] Tailwind config exposes all tokens required by `DESIGN.md`

**Commit:** `phase(0): project scaffold, Makefile, docker, web boilerplate`

---

## Phase 1 — Config & Database Connection

**Goal:** Implement config loading and verified PostgreSQL connection.

**Tasks:**
1. Install Go deps: `cobra`, `chi`, `pgx/v5`, `pgxpool`, `sqlx`, `yaml.v3`, `fatih/color`, `go-sqlite3`
2. Implement `internal/config/config.go`:
   - Load from `rift.yaml` (via `gopkg.in/yaml.v3`)
   - Override with env vars: `RIFT_DATABASE_URL`, `RIFT_ENV`, `RIFT_MIGRATIONS_DIR`, `RIFT_PORT`, `RIFT_TOKEN`
   - Exported `Config` struct matching fields in PRD
3. Implement `internal/db/db.go`:
   - `NewPool(ctx, cfg) (*pgxpool.Pool, error)` using `pgxpool.New`
   - `Ping(ctx, pool)` for connection health check
4. Implement Cobra root command in `cmd/rift/main.go`:
   - Subcommands registered: `new`, `up`, `down`, `status`, `diff`, `server`, `lint`
   - `--config` flag defaulting to `./rift.yaml`
   - Global `--verbose` flag
5. Implement `rift config check` subcommand: loads config, pings DB, prints colored OK/FAIL status

**Verification:**
- [ ] `./rift --help` shows all subcommands
- [ ] `./rift config check` connects to a real PG instance and prints green OK
- [ ] Invalid `DATABASE_URL` prints clear error message, exit code 1
- [ ] Unit test: config loads from YAML and env vars with correct precedence

**Commit:** `phase(1): config loading, db pool, cobra root, config-check command`

---

## Phase 2 — Migration State Table & Core Runner

**Goal:** Bootstrap the `_rift_migrations` table and implement the core apply/rollback engine with advisory locks.

**Tasks:**
1. Implement `internal/migration/state.go`:
   - `EnsureStateTable(ctx, pool)` — creates `_rift_migrations` if not exists (idempotent)
   - `GetApplied(ctx, pool) ([]MigrationRecord, error)`
   - `RecordApplied(ctx, tx, record MigrationRecord) error`
   - `RecordRolledBack(ctx, tx, version string) error`
2. Implement `internal/migration/checksum.go`:
   - `Checksum(content []byte) string` — SHA-256 hex
3. Implement `internal/migration/loader.go`:
   - `LoadFiles(dir string) ([]MigrationFile, error)` — reads `*.up.sql` / `*.down.sql` pairs, sorted by version timestamp
   - `MigrationFile` struct: `Version`, `Filename`, `UpSQL`, `DownSQL`, `Checksum`
4. Implement `internal/migration/conflict.go`:
   - `DetectConflicts(applied []MigrationRecord, files []MigrationFile) []Conflict`
   - Conflict types: `MISSING_FILE` (applied in DB but no local file), `CHECKSUM_MISMATCH`
5. Implement `internal/migration/runner.go`:
   - `AcquireAdvisoryLock(ctx, pool) (release func(), err error)` using `pg_try_advisory_lock(hashtext('rift_migration_lock'))`
   - `RunUp(ctx, pool, cfg, dryRun bool) error` — applies pending migrations in order, records each in `_rift_migrations`
   - `RunDown(ctx, pool, cfg, steps int) error` — rolls back N migrations in reverse order
   - Each migration runs in a DDL transaction; if it fails, the transaction is rolled back and a clear error is printed

**Verification:**
- [ ] `EnsureStateTable` is idempotent (run twice, no error)
- [ ] `LoadFiles` correctly parses and sorts 5 test migration files
- [ ] Unit test: conflict detection catches `MISSING_FILE` and `CHECKSUM_MISMATCH`
- [ ] Integration test: `RunUp` applies 3 test migrations, all appear in `_rift_migrations`
- [ ] Advisory lock: a second concurrent `RunUp` call returns a clear "migration already in progress" error

**Commit:** `phase(2): migration state table, runner, advisory locks, conflict detection`

---

## Phase 3 — CLI Commands: new, up, down, status

**Goal:** Implement the four primary CLI commands backed by Phase 2 internals.

**Tasks:**
1. Implement `internal/cli/new.go` → `rift new <name>`:
   - Generates `migrations/{timestamp}_{name}.up.sql` and `migrations/{timestamp}_{name}.down.sql`
   - Timestamp format: `YYYYMMDD_HHmmss`
   - Writes a comment header: `-- Migration: {name} | Created: {datetime}`
   - Prints file paths in green using `fatih/color`
2. Implement `internal/cli/up.go` → `rift up [--dry-run] [--force]`:
   - Loads config, pings DB, ensures state table
   - Checks for conflicts (halt on conflict unless `--force`)
   - Runs linter on pending migrations, prints warnings
   - Calls `runner.RunUp`
   - Prints each applied migration with timing in ms
3. Implement `internal/cli/down.go` → `rift down [--steps N]`:
   - Prompts for confirmation: `"Roll back 1 migration? (yes/no): "`
   - Calls `runner.RunDown`
4. Implement `internal/cli/status.go` → `rift status`:
   - Renders a table: `VERSION | FILENAME | STATUS | APPLIED_AT | APPLIED_BY | TIME_MS`
   - Applied migrations in green, pending in yellow, rolled-back in gray

**Verification:**
- [ ] `rift new add_users_table` creates correctly named file pair
- [ ] `rift status` shows correct applied/pending split against real DB
- [ ] `rift up --dry-run` prints what would be applied without modifying DB
- [ ] `rift down` prompts for confirmation; entering "no" aborts
- [ ] `rift up` halts with clear message on conflict detection

**Commit:** `phase(3): rift new, up, down, status CLI commands`

---

## Phase 4 — Schema Diff Engine

**Goal:** Implement the schema introspection and diff computation engine — the project's signature feature.

**Tasks:**
1. Implement `internal/diff/introspect.go`:
   - `IntrospectLive(ctx, pool, schema string) (*SchemaSnapshot, error)`
   - Uses `information_schema.tables`, `information_schema.columns`, `pg_indexes`, `pg_constraint` to build a complete snapshot of: tables, columns (name, type, nullable, default), indexes, foreign keys, unique constraints
2. Implement `internal/diff/parse.go`:
   - `ParseMigrationSQL(sql string) (*SchemaSnapshot, error)` — parse `CREATE TABLE`, `ALTER TABLE ADD/DROP/MODIFY COLUMN`, `CREATE INDEX`, `DROP TABLE` statements from SQL text to build an expected schema snapshot
   - Start with regex + AST-lite parsing for the most common DDL patterns; document known limitations
3. Implement `internal/diff/diff.go`:
   - `ComputeDiff(before, after *SchemaSnapshot) *SchemaDiff`
   - `SchemaDiff` struct: `TablesAdded`, `TablesDropped`, `TablesModified` (with `ColumnsAdded`, `ColumnsModified`, `ColumnsDropped`), `IndexChanges`, `ConstraintChanges`
   - `RenderTerminal(diff *SchemaDiff)` — colored CLI output using `fatih/color`
   - `RenderJSON(diff *SchemaDiff) ([]byte, error)` — for API response
4. Implement `internal/cli/diff.go` → `rift diff`:
   - Loads pending migrations, computes diff against live DB, prints colored terminal output
   - Summary line: `"3 tables modified, 1 table added, 2 indexes changed"`

**Verification:**
- [ ] `IntrospectLive` correctly reflects a 5-table test schema including indexes and FK constraints
- [ ] `ParseMigrationSQL` correctly parses CREATE TABLE with 8 columns, ALTER TABLE ADD COLUMN, ALTER TABLE DROP COLUMN, CREATE INDEX
- [ ] `ComputeDiff` correctly identifies: 1 new table, 1 dropped column, 1 modified column type, 1 new index across a test migration pair
- [ ] `rift diff` prints colored output and exits 0 when no diff, exits 1 when diff exists (for CI use)
- [ ] Unit tests cover all 4 diff categories with table-driven test cases

**Commit:** `phase(4): schema diff engine — introspect, parse, diff, CLI command`

---

## Phase 5 — Migration Linter

**Goal:** Implement the DDL pattern linter that flags dangerous operations.

**Tasks:**
1. Implement `internal/linter/linter.go`:
   - `LintSQL(sql string) []LintWarning`
   - `LintWarning` struct: `Pattern string`, `Line int`, `Severity string` (`"error"` or `"warning"`), `Message string`, `Suggestion string`
   - Patterns to detect (from PRD):
     1. `DROP COLUMN` → error
     2. `RENAME COLUMN` → error
     3. `ALTER COLUMN ... NOT NULL` without `DEFAULT` → error
     4. `DROP TABLE` → error
     5. `CREATE INDEX` without `CONCURRENTLY` → warning
     6. `ADD CONSTRAINT ... FOREIGN KEY` without `NOT VALID` → warning
2. Implement `internal/cli/lint.go` → `rift lint [file]`:
   - Lints all pending migrations (or a specific file)
   - Prints warnings with line numbers and suggestions
   - Exit code 1 if any errors found (unless `--warn-only` in config)
3. Integrate linter into `runner.RunUp`: print warnings before applying, block on errors unless `--force`
4. Add linter summary to `rift diff` output

**Verification:**
- [ ] Unit test: each of the 6 patterns is correctly detected in isolation
- [ ] `rift lint` on a file with `DROP COLUMN` exits 1 and prints the suggestion
- [ ] `rift lint` on a clean file exits 0
- [ ] `rift up --force` applies despite linter errors and prints a warning banner

**Commit:** `phase(5): DDL linter — 6 dangerous patterns, CLI command, runner integration`

---

## Phase 6 — Go API Server

**Goal:** Implement the Chi REST API server that backs the web dashboard.

**Tasks:**
1. Implement `internal/api/server.go`:
   - `NewServer(cfg *config.Config, pool *pgxpool.Pool) *chi.Mux`
   - Bearer token middleware: reads the Authorization header as a Bearer token and returns 401 if missing or mismatched
   - CORS middleware for local dev (`localhost:5173`)
   - Request logging middleware
2. Implement all API handlers in `internal/api/handlers/`:
   - `GET /api/v1/status` → env name, counts (applied/pending/rolled-back), last deployed
   - `GET /api/v1/migrations` → full list with status
   - `GET /api/v1/migrations/:version` → single migration + SQL content
   - `GET /api/v1/migrations/:version/diff` → schema diff JSON for that migration
   - `POST /api/v1/migrate/up` → trigger `RunUp`; stream output via SSE (`text/event-stream`)
   - `POST /api/v1/migrate/down` → trigger `RunDown` (body: `{"steps": 1}`)
   - `GET /api/v1/history` → audit log (all records from `_rift_migrations`, newest first)
   - `GET /api/v1/lint` → lint all pending migrations, return structured warnings
   - `GET /api/v1/team` → team members list from config
   - `GET /api/v1/conflicts` → current conflict detection report
3. Implement SSE for `POST /api/v1/migrate/up`:
   - Stream each migration's apply event as it happens
   - Final event: `{status: "done", applied: N}` or `{status: "error", message: "..."}`
4. Implement `internal/cli/server.go` → `rift server`:
   - Starts Chi server on configured port
   - Serves embedded React UI from `go:embed internal/embed/ui`
   - Prints startup message with URL

**Verification:**
- [ ] `GET /api/v1/status` returns correct JSON with valid counts
- [ ] `GET /api/v1/migrations` returns correct applied/pending split
- [ ] Bearer token middleware returns 401 for missing/wrong token
- [ ] `POST /api/v1/migrate/up` SSE stream sends one event per migration applied
- [ ] `GET /api/v1/team` and `GET /api/v1/conflicts` return JSON consumed by the Team page
- [ ] `rift server` starts and serves a 200 on `/`

**Commit:** `phase(6): Chi API server, all REST endpoints, SSE for live deploy, embed UI mount`

---

## Phase 7 — React Web Dashboard: Foundation & Migration List

**Goal:** Build the React app shell, routing, API client, and the Migrations list page.

**Tasks:**
1. Set up `react-router-dom` with routes: `/`, `/migrations`, `/migrations/:version`, `/migrations/:version/diff`, `/team`, `/settings`
2. Set up TanStack Query `QueryClient` in `main.tsx`
3. Set up Zustand store for: auth token, environment name, sidebar open state
4. Build shared components:
   - `AppShell` — fixed sidebar + sticky top app bar matching `DESIGN.md` Sections 7.1 and 7.2
   - `Sidebar` — exact nav structure and active right-border accent from `DESIGN.md` Section 12
   - `StatusBadge` — applied, pending, pending-danger, failed, rolled-back chips matching `DESIGN.md` Section 7.3
   - `StatCard`, `DataTable`, `QuickActionsCard`, `LinterAlertsCard`, `LoadingSkeleton`, `ErrorBoundary`
5. Build API client layer (`src/lib/api.ts`):
   - `fetchMigrations()`, `fetchMigration(version)`, `fetchDiff(version)`, `fetchStatus()`, `fetchHistory()`, `fetchLint()`, `fetchTeam()`, `fetchConflicts()`
   - `triggerUp()` (returns EventSource), `triggerDown(steps)`
   - All requests attach the configured Bearer token from Zustand store
6. Build `/` as a redirect to `/migrations` or render the same Migration Dashboard.
7. Build `/migrations` Migration Dashboard from `DESIGN.md` Section 8.1:
   - Three stat cards: Total Migrations, Applied, Pending
   - Recent Activity table: Status | ID | Name | Author | Applied Date | Actions
   - Search input for name/version filtering
   - Right sidebar with Quick Actions and Linter Alerts cards
   - Row View action → `/migrations/:version`; Diff action → `/migrations/:version/diff`
8. Build `/migrations/:version` SQL Authoring Interface from `DESIGN.md` Section 8.3:
   - Left schema/table browser, center editable CodeMirror SQL editor, right Zero-Downtime Linter panel
   - Metadata strip: filename input, category dropdown, author badge
   - Table browser insertion action inserts a table name at the active editor cursor

**Verification:**
- [ ] App loads on `http://localhost:5173` with correct token prompt if not set
- [ ] `/migrations` renders the mockup-aligned dashboard layout, stat cards, table, Quick Actions, and Linter Alerts
- [ ] Click a migration → SQL Authoring Interface opens with CodeMirror syntax highlighting and linter panel
- [ ] Sidebar links navigate correctly and use the `DESIGN.md` active border treatment

**Commit:** `phase(7): React app shell, routing, API client, dashboard, migrations list`

---

## Phase 8 — Schema Diff Viewer UI & Team Page

**Goal:** Build the signature schema diff viewer and the Team & Deployment History page from the design mockups.

**Tasks:**
1. Build `/migrations/:version/diff` page from `DESIGN.md` Section 8.2:
   - Calls `GET /api/v1/migrations/:version/diff`
   - Renders a custom full-height split-pane viewer: left `LOCAL MIGRATIONS`, right `LIVE DATABASE`
   - Summary/control panel: change counts plus `SAFE PREVIEW (DDL)` toggle
   - SQL rows use design-specified line numbers, syntax colors, addition backgrounds, deletion backgrounds, and strikethroughs
   - `APPLY MIGRATIONS` button → opens confirmation modal → calls `POST /api/v1/migrate/up` and displays SSE progress inline
2. Build Confirmation modal component:
   - Shows migration filename, diff summary, linter warnings
   - "Apply" (primary) / "Cancel" (secondary) buttons
   - Disabled "Apply" button if linter has errors and no `--force` equivalent toggle
3. Build reusable SSE deploy log panel shown inside the diff confirmation/apply flow:
   - Real-time terminal-style output as migrations apply
   - Log lines styled as timestamp + migration name + status icon
   - Final status card: `N migrations applied in Xms` or error message
4. Build `/team` page from `DESIGN.md` Section 8.4:
   - Conflict Detection card from `GET /api/v1/conflicts`
   - Deployment History timeline from `GET /api/v1/history`
   - Team Access panel from `GET /api/v1/team`
   - Notifications/Webhooks panel with Slack/Discord toggles and URL fields saved as config-only UI

**Verification:**
- [ ] Diff viewer shows correct local/live panes for a test migration
- [ ] Diff viewer matches `DESIGN.md` split-pane headers, syntax colors, and line treatments
- [ ] Linter warnings render with suggestion text in the apply confirmation flow
- [ ] SSE apply stream shows live log lines as they arrive and completes with success/error state
- [ ] Team page shows conflicts, deployment timeline, team members, and webhook fields

**Commit:** `phase(8): schema diff viewer UI, SSE apply flow, team deployment page`

---

## Phase 9 — Settings Page, Single Binary Embed & Docker

**Goal:** Complete the settings UI, finalize `go:embed` integration, and validate Docker deployment.

**Tasks:**
1. Build `/settings` page:
   - Form fields: API token (masked, reveal on click), environment name
   - Save updates Zustand store + `localStorage`
   - Database connection status indicator (calls `GET /api/v1/status`)
2. Finalize `go:embed` integration:
   - `make build-web` copies `web/dist/*` to `internal/embed/ui/`
   - `//go:embed internal/embed/ui` directive in `internal/api/server.go`
   - Serve SPA correctly: all non-`/api/*` routes return `index.html`
3. Validate single binary:
   - `make build` produces `./rift` binary ~20MB or less
   - `./rift server` serves both API and UI from the single binary
   - No external files required at runtime
   - Direct navigation to `/migrations`, `/migrations/:version`, `/migrations/:version/diff`, `/team`, and `/settings` returns the SPA
4. Write `docker/Dockerfile`:
   - Stage 1: `node:20-alpine` → build React (`npm run build`)
   - Stage 2: `golang:1.22-alpine` → build Go binary with embedded UI
   - Stage 3: `alpine:3.19` → copy binary, expose port, `ENTRYPOINT ["./rift", "server"]`
5. Write `docker-compose.yml`:
   - `rift` service built from `./docker/Dockerfile`
   - `postgres:16` service with volume
   - Environment variables for connection
6. Test full Docker deployment:
   - `docker compose up` → navigate to `http://localhost:7878`
   - Verify dashboard, SQL authoring interface, diff viewer, team page, and settings page all work from Docker

**Verification:**
- [ ] `make build` produces a single binary with embedded UI
- [ ] `./rift server` serves UI at `localhost:7878` and API at `localhost:7878/api/v1/`
- [ ] `docker compose up` works end-to-end against a fresh postgres container
- [ ] Settings page saves token and persists across page refresh
- [ ] Direct SPA routes for `/team` and `/migrations/:version/diff` work when served from the binary

**Commit:** `phase(9): settings UI, go:embed single binary, Dockerfile, docker-compose validation`

---

## Phase 10 — Tests, README & Polish

**Goal:** Add comprehensive tests, write the final README with quickstart, and polish the UI/CLI for portfolio presentation.

**Tasks:**
1. Go tests:
   - `internal/migration/` — unit tests for loader, checksum, conflict detection, state table
   - `internal/diff/` — table-driven unit tests for introspect, parse, diff (all 4 change types)
   - `internal/linter/` — unit test for each of the 6 lint patterns
   - `internal/api/` — handler integration tests using `httptest.NewRecorder`
2. React tests (Vitest + Testing Library):
   - `MigrationList` renders correct row count from mocked API
   - `StatusBadge` renders correct color for each status
   - `DiffViewer` renders `LOCAL MIGRATIONS` and `LIVE DATABASE` panes from mock diff data
   - `TeamPage` renders conflict, deployment timeline, team access, and webhook sections from mocked API data
3. Write `README.md`:
   - Badges: Go version, license, last commit
   - One-liner and "Why Rift?" section
   - 5-minute quickstart with Docker Compose
   - CLI reference table for all commands and flags
   - Screenshot placeholders for: dashboard, SQL authoring interface, diff viewer, team deployment history
   - Architecture diagram (ASCII)
   - Contributing guide
4. CLI polish:
   - `rift --version` prints version + build commit hash (injected via ldflags)
   - All error messages use `fatih/color` red + clear fix instructions
   - Progress spinner during `rift up` using a simple goroutine + ticker
5. UI polish:
   - Empty states for all pages (no migrations, no history)
   - Error states with retry buttons
   - Loading skeletons for migration list and diff viewer
   - Dark mode toggle (Tailwind `dark:` classes + localStorage)

**Verification:**
- [ ] `go test ./...` passes with 0 failures
- [ ] `cd web && npm run test` passes with 0 failures
- [ ] `./rift --version` prints correct version
- [ ] README quickstart works end-to-end from a fresh clone
- [ ] Empty state renders on migrations page when DB has no migrations
- [ ] UI remains visually consistent with `DESIGN.md` across dashboard, authoring, diff, team, and settings routes

**Commit:** `phase(10): tests, README, CLI polish, UI empty states, dark mode`
