# PRD — Rift: Self-Hosted PostgreSQL Migration Manager

> **Design reference:** All frontend visual decisions — color tokens, typography, component patterns, and page layouts — are specified in [`DESIGN.md`](./DESIGN.md). The mockups in the design package are the authoritative visual reference. This PRD defines *what* the product does and *how* the backend works. DESIGN.md defines *how it looks*.

## Overview

Rift is a self-hosted, open-source database migration manager for PostgreSQL teams. It combines a developer-first CLI (think Flyway or golang-migrate) with a React-powered web dashboard to author, preview, apply, and audit SQL migrations collaboratively. The killer feature is the **schema diff viewer** — before applying migrations, Rift compares what the SQL files describe against the live database and renders an exact, human-readable diff of what will change.

**One-liner:** Flyway + a beautiful React UI — write migrations locally, preview schema diffs in a dashboard, coordinate with your team, and never be surprised by a deployment again.

---

## Problem Statement

Database migrations are one of the highest-risk operations in any engineering team's workflow:

- **Blind deployments:** Developers apply `ALTER TABLE` without seeing what the current live schema looks like. "I thought that column already existed."
- **Team conflicts:** Two developers branch off `main`, each write migrations, and merge — producing conflicting or duplicated state in the `_migrations` table.
- **No rollback confidence:** Most teams lack a fast path to inspect, confirm, and roll back a failed migration in production.
- **Dangerous DDL patterns:** `DROP COLUMN`, `RENAME COLUMN`, adding `NOT NULL` without a default — these are applied without any tooling-level warning.
- **Audit gaps:** No record of who applied what migration, when, and from which environment.

Existing tools (Flyway, Liquibase, golang-migrate) solve the core sequencing problem but offer no diff preview, no modern UI, no team conflict detection, and no linting for dangerous patterns.

---

## Target Users

| Persona | Description |
|---|---|
| **Solo developer** | Uses Rift CLI as a smarter golang-migrate; browses history in the UI |
| **Small eng team (2–10)** | Needs team coordination, conflict detection, per-environment dashboards |
| **DevOps/DBA** | Reviews deployment history, audit logs, and gets pre-deployment diff reports before approving |
| **Open-source contributor** | Attracted by a useful, well-documented Go + React project with real PostgreSQL depth |

---

## Goals & Non-Goals

### Goals
- CLI-first developer experience with `rift new`, `rift up`, `rift down`, `rift diff`, `rift status`
- Schema diff viewer showing exact DDL changes before any migration is applied
- Web dashboard for migration history, SQL preview, rollback, and deployment audit
- Team-safe migration state stored in PostgreSQL (`_rift_migrations` table) with conflict detection
- Advisory lock usage to prevent concurrent migration runs
- Built-in linter flagging dangerous DDL patterns with safer alternatives
- Single binary deployment with embedded React UI (`go:embed`)
- Docker support for self-hosted server mode
- Deployment readiness checks with `rift config doctor` for public/demo environments
- SQLite for local config/state when running in CLI-only mode

### Non-Goals (v1)
- Multi-database support (MySQL, SQLite targets) — PostgreSQL only
- GitHub/GitLab CI integration — out of scope for v1
- Elasticsearch / full-text search across migration history
- RBAC / OAuth — authentication is deferred; single shared token for team mode
- Automatic rollback on deploy failure — manual rollback only in v1

---

## Core Features

### 1. Migration Authoring CLI

```
rift new add_users_table
# → migrations/20240601_143201_add_users_table.up.sql
# → migrations/20240601_143201_add_users_table.down.sql

rift up                 # apply all pending migrations
rift up --dry-run       # preview what would be applied
rift down               # roll back last applied migration (with confirmation prompt)
rift down --steps 3     # roll back last 3 migrations
rift status             # show applied/pending migration list
rift diff               # compare migration files against live DB schema
rift config doctor      # check deployment readiness without printing secrets
```

Each migration file pair (`up.sql` / `down.sql`) uses a timestamp prefix for strict ordering. The CLI reads config from `rift.yaml` or environment variables.

### 2. Schema Diff Viewer

The diff engine works in two steps:

1. **Parse** the migration SQL files to reconstruct the expected schema state after all pending migrations are applied.
2. **Introspect** the live PostgreSQL database via `information_schema` and `pg_catalog` to get the current schema.
3. **Diff** the two representations and produce a structured list of changes: tables added/dropped, columns added/modified/dropped, indexes changed, constraints changed.

The web dashboard renders this diff with the custom split-pane SQL viewer specified in `DESIGN.md` Section 8.2. The CLI (`rift diff`) renders colored terminal output using `fatih/color`.

**This is the feature that will attract the most attention** — no other self-hosted OSS migration tool does this before apply.

### 3. Team Collaboration & Conflict Detection

Migration state is stored in a `_rift_migrations` table in the target PostgreSQL database:

```sql
CREATE TABLE _rift_migrations (
  id            SERIAL PRIMARY KEY,
  version       TEXT NOT NULL UNIQUE,    -- timestamp prefix
  filename      TEXT NOT NULL,
  checksum      TEXT NOT NULL,           -- SHA-256 of up.sql content
  applied_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  applied_by    TEXT,                    -- hostname or user from rift.yaml
  execution_ms  INTEGER,
  rolled_back   BOOLEAN NOT NULL DEFAULT false
);
```

Conflict detection: if Rift finds a migration in `_rift_migrations` that is **not present** in the local migration folder, or finds a migration file whose checksum differs from the stored checksum, it halts with an explicit conflict report and resolution instructions.

Advisory locks (`pg_try_advisory_lock(hashtext('rift_migration_lock'))`) prevent two concurrent `rift up` runs from the same or different machines.

### 4. Web Dashboard

The dashboard is a dark-themed, developer-focused SPA. The visual design system is fully specified in `DESIGN.md` — refer to it for all colors, typography, component patterns, and page layouts. Below is the functional specification for each route.

**Routes:**

`/migrations` — **Migration Dashboard** (default landing page)
- Three stat cards across the top: Total Migrations, Applied (emerald), Pending (with electric-blue pulsing glow when > 0)
- "Recent Activity" table: STATUS | ID (Timestamp) | NAME | AUTHOR | APPLIED DATE | ACTIONS
- Status column uses colored chips: Applied (emerald), Pending (neutral), Pending+Linter-Error (red-tinted with warning icon), Failed (red)
- Right sidebar: "Quick Actions" card (Connect to DB, Sync Local Files) + "Linter Alerts" card showing any dangerous pending migrations
- Search input for filtering the migration table by name or version

`/migrations/:version` — **SQL Authoring Interface**
- Three-panel layout: left table browser (schema tree of live DB tables), center SQL editor (CodeMirror 6 with JetBrains Mono + SQL syntax highlighting), right linter panel
- Metadata strip above editor: editable filename, category dropdown, author badge
- Linter panel shows real-time issues with red left-accent cards and "Auto-Fix" suggestion buttons
- Passing linter rules shown with green dots below the error cards

`/migrations/:version/diff` — **Schema Diff Viewer** (the signature feature)
- Full-height split-pane layout: left pane = "LOCAL MIGRATIONS" (desired state from SQL files), right pane = "LIVE DATABASE" (current state from pg_catalog)
- Summary bar above panes: "3 tables to be created · 1 column to be altered" with counts in emerald/amber
- "SAFE PREVIEW (DDL)" toggle for DDL-transaction safe mode
- Added lines highlighted in emerald (`bg-secondary/5`), removed lines in red with strikethrough (`bg-error/5`)
- "APPLY MIGRATIONS" primary button in top-right; triggers confirmation modal

`/team` — **Team & Deployment**
- Conflict Detection card: shown prominently at top when conflicts exist; displays two conflicting branch snippets side by side with "Resolve manually" CTA
- Deployment History Timeline: vertical timeline with success (emerald dot) and failed (red dot) entries; failed entries show inline error message in monospace
- Right column: Team Access panel (avatar + name + role badge per member) + Notifications panel (Slack/Discord webhook toggles)

`/settings` — **Settings**
- Database connection status indicator
- API token field (masked, reveal toggle)
- Environment name (from `GET /api/v1/status`)
- Persists to `localStorage` and Zustand store

The Go API server exposes a REST API (`/api/v1/...`) served by Chi. The React SPA is embedded in the binary via `go:embed`.

### 5. Zero-Downtime Migration Linter

Built into `rift up`, `rift diff`, and the web dashboard. Before applying, Rift scans the pending migration SQL and flags:

| Pattern | Risk | Suggestion |
|---|---|---|
| `DROP COLUMN` | Data loss | Add deprecation comment, use soft-delete pattern |
| `RENAME COLUMN` | Breaking change for running apps | Add new column, migrate data, drop old |
| `ALTER COLUMN ... NOT NULL` without DEFAULT | Table lock / existing nulls | Set DEFAULT first, backfill, then add constraint |
| `DROP TABLE` | Data loss | Rename to `_deprecated_*` first |
| `CREATE INDEX` without `CONCURRENTLY` | Table lock | Use `CREATE INDEX CONCURRENTLY` |
| `ADD CONSTRAINT ... FOREIGN KEY` | Full table scan lock | Add constraint as `NOT VALID`, then `VALIDATE CONSTRAINT` |

Linter output is shown as warnings (not hard blocks) with a `--force` override flag.

---

## Technical Architecture

### Directory Structure

```
rift/
├── cmd/
│   └── rift/
│       └── main.go               # Cobra root command
├── internal/
│   ├── cli/                      # CLI command handlers
│   │   ├── new.go
│   │   ├── up.go
│   │   ├── down.go
│   │   ├── status.go
│   │   └── diff.go
│   ├── migration/                # Core migration engine
│   │   ├── runner.go             # Apply/rollback logic with advisory locks
│   │   ├── state.go              # _rift_migrations table read/write
│   │   ├── conflict.go           # Conflict detection logic
│   │   └── checksum.go
│   ├── diff/                     # Schema diff engine
│   │   ├── introspect.go         # pg_catalog / information_schema queries
│   │   ├── parse.go              # SQL file → expected schema AST
│   │   └── diff.go               # Diff computation → structured output
│   ├── linter/
│   │   └── linter.go             # DDL pattern analysis
│   ├── api/                      # Chi HTTP server
│   │   ├── server.go
│   │   ├── routes.go
│   │   └── handlers/
│   ├── config/
│   │   └── config.go             # rift.yaml + env var loading
│   └── embed/
│       └── ui/                   # go:embed target (built React app)
├── web/                          # React/TypeScript frontend
│   ├── src/
│   │   ├── pages/
│   │   ├── components/
│   │   ├── stores/               # Zustand stores
│   │   ├── hooks/                # TanStack Query hooks
│   │   └── lib/
│   ├── package.json
│   └── vite.config.ts
├── migrations/                   # Example/bootstrap migrations for _rift_migrations itself
├── docker/
│   └── Dockerfile
├── rift.yaml.example
├── Makefile
└── README.md
```

### Go Dependencies

| Package | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI command framework |
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/jackc/pgx/v5` | PostgreSQL driver |
| `github.com/jackc/pgx/v5/pgxpool` | Connection pool |
| `github.com/jmoiron/sqlx` | Schema introspection queries |
| `github.com/golang-migrate/migrate/v4` | Migration sequencing library |
| `github.com/fatih/color` | Colored terminal output |
| `github.com/mattn/go-sqlite3` | SQLite for local config mode |
| `gopkg.in/yaml.v3` | Config file parsing |
| `crypto/sha256` (stdlib) | Migration checksum |

### React/TypeScript Dependencies

| Package | Purpose |
|---|---|
| `react`, `react-dom` | UI framework |
| `@tanstack/react-query` | Server state management |
| `zustand` | Client UI state |
| `tailwindcss` | Utility-first styling (see `DESIGN.md` for full token config) |
| `@tailwindcss/forms` | Form element reset styles |
| `@tailwindcss/container-queries` | Container query plugin |
| `@uiw/react-codemirror` + `@codemirror/lang-sql` | SQL editor with custom dark theme (JetBrains Mono, `surface-container-lowest` bg) |
| `@codemirror/language` | CodeMirror HighlightStyle for SQL syntax colors |
| `react-router-dom` | Client-side routing |
| `vitest` + `@testing-library/react` | Unit/integration tests |

> **Icons:** Use **Material Symbols Outlined** (variable font via Google Fonts CDN), not lucide-react. See `DESIGN.md` Section 10 for the full icon mapping.

> **Fonts:** Load Geist, Inter, and JetBrains Mono from Google Fonts. See `DESIGN.md` Section 3.

### Config File (`rift.yaml`)

```yaml
environment: production
database_url: postgres://user:pass@localhost:5432/mydb
migrations_dir: ./migrations
author: fandy@example.com   # stored in _rift_migrations.applied_by

server:
  port: 7878
  token: your-secret-token  # Bearer token for API access

linter:
  warn_only: false           # set true to downgrade errors to warnings
```

---

## API Endpoints (v1)

```
GET    /api/v1/status                    → environment name, counts (applied/pending/rolledback), last deployed
GET    /api/v1/migrations                → full list with status, author, timing, linter flag
GET    /api/v1/migrations/:version       → single migration detail + up.sql + down.sql content
GET    /api/v1/migrations/:version/diff  → structured schema diff JSON for that migration
POST   /api/v1/migrate/up               → apply all pending (SSE stream, one event per migration)
POST   /api/v1/migrate/down             → roll back N migrations (body: {"steps": 1})
GET    /api/v1/history                  → deployment audit log, newest first
GET    /api/v1/lint                     → lint all pending migrations, return structured warnings
GET    /api/v1/team                     → team members list (from rift.yaml team config)
GET    /api/v1/conflicts                → detect and return any migration conflicts
```

---

## Data Models

### MigrationRecord (Go struct / DB row)
```go
type MigrationRecord struct {
    ID          int
    Version     string    // "20240601_143201"
    Filename    string    // "20240601_143201_add_users_table"
    Checksum    string    // SHA-256 of up.sql
    AppliedAt   time.Time
    AppliedBy   string
    ExecutionMs int
    RolledBack  bool
}
```

### SchemaDiff (structured diff output)
```go
type SchemaDiff struct {
    TablesAdded    []TableDef
    TablesDropped  []TableDef
    TablesModified []TableModification
    IndexChanges   []IndexChange
    ConstraintChanges []ConstraintChange
}
```

---

## Success Metrics

- CLI: `rift up` and `rift diff` are the primary entry points; zero-config from `rift.yaml`
- Web dashboard loads in < 500ms on local network and matches the `DESIGN.md` visual system on the four mockup-backed routes
- Diff viewer correctly identifies 100% of schema changes for the 6 most common DDL operations
- Linter catches all 6 defined dangerous patterns
- Single binary (`go build`) embeds the full React UI with no external files needed
- README includes a working 5-minute quickstart (Docker Compose)

---

## Design Assets

| File | Purpose |
|---|---|
| `DESIGN.md` | Full design system: color tokens, typography, component specs, page-by-page layout |
| `doc/rift_dashboard.png` + `doc/rift_dashboard.html` | Migration Dashboard reference screenshot and HTML mockup source |
| `doc/schema_diff_viewer.png` + `doc/schema_diff_viewer.html` | Schema Diff Viewer reference screenshot and HTML mockup source |
| `doc/sql_authoring_interface.png` + `doc/sql_authoring_interface.html` | SQL Authoring Interface reference screenshot and HTML mockup source |
| `doc/team_deployment_history.png` + `doc/team_deployment_history.html` | Team & Deployment History reference screenshot and HTML mockup source |

The HTML mockup source files in `doc/` contain the exact layout intent and Tailwind class combinations used in the design package. `DESIGN.md` remains the normalized implementation reference; when PRD page descriptions and design details differ, implement `DESIGN.md`.

---

## Out of Scope (v2+)
- Multi-tenant SaaS hosting
- GitHub Actions / GitLab CI pipeline integration
- MySQL / SQLite migration target support
- Auto-rollback on deploy failure
- RBAC and user accounts

> **Note:** Slack/Discord webhook configuration UI is shown in the `team_deployment_history` mockup as a v1 UI element (toggle + URL input), but the actual webhook *delivery* implementation is v2+. In v1, the UI renders the fields but saves them to `rift.yaml` only — no HTTP delivery logic.