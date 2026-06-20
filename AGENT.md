# AGENT.md — Rift Autonomous Build Agent

## Role

You are an autonomous software engineering agent. Your task is to build **Rift** — a self-hosted PostgreSQL migration manager — according to `PRD.md`, `DESIGN.md`, and `BUILD_PHASES.md`. You work one phase at a time, verify each coherent step before committing, and push to Git after every small verified step so changes are easy to track and revert.

You do not ask for permission between tasks. You do not pause for confirmation unless an unrecoverable ambiguity blocks forward progress. You work until the project is complete.

---

## Mandatory Reading

Before writing any code, read these files in full:

1. `PRD.md` — product requirements, architecture, data models, API spec
2. `DESIGN.md` — authoritative frontend visual system, page layouts, mockup-derived components
3. `BUILD_PHASES.md` — phase breakdown, task lists, verification checklists

If any mandatory file is missing, halt and report.

---

## Execution Protocol

### Starting a Phase

1. Read the phase from `BUILD_PHASES.md`.
2. For frontend or API/UI contract work, re-read the relevant `DESIGN.md` sections before editing.
3. Execute tasks as small, coherent, independently revertible steps.
4. Run the narrowest relevant verification for each step.
5. Commit and push after every verified step, without asking for permission.
6. Run the full phase verification checklist before the phase milestone commit — **every item must pass**.
7. Commit and push the phase milestone.

### Small-Step Commit Format

Use this after each small verified step:

```bash
git add -A
git commit -m "<type>(<scope>): <small verified change>" \
  -m "- Verified with <command>" \
  -m "- Revert boundary: <files or subsystem>"
git push origin main
```

A small step should contain one coherent change: one backend endpoint, one migration runner behavior, one frontend page slice, one bug fix with regression coverage, or one docs/config update. Do not batch unrelated work into a single commit.

If `git push` fails, halt and report the exact error. Do not continue accumulating unpushed changes.

### Phase Completion Commit Format

```
git add -A
git commit -m "phase(N): <short description from BUILD_PHASES>"
git push origin main
```

Never leave verified work uncommitted. Never skip a phase. Never combine phases into one commit.

### Blocked Phase

If a phase fails its verification checklist:
1. Diagnose the failure.
2. Fix the root cause.
3. Re-run verification.
4. Only commit once all checks pass.

If the failure cannot be resolved (e.g., a missing dependency that does not exist), report the blocker with a specific error message and proposed resolution.

---

## Environment Assumptions

- Go 1.22+ is installed and `go` is on PATH
- Node.js 20+ and `npm` are installed
- Docker and `docker compose` are available
- A PostgreSQL 15/16 instance is accessible for integration tests; connection string is in `RIFT_DATABASE_URL` env var or `rift.yaml`
- Git is initialized with a remote named `origin`; `main` is the default branch
- SSH key is configured for GitHub push (no password prompts)

---

## Code Standards

### Go

- All exported functions have godoc comments.
- Errors are always wrapped with context: `fmt.Errorf("loading migration files: %w", err)`
- No `panic()` in library code; only in `main()` for unrecoverable startup failures.
- Use `context.Context` as the first argument for all functions that do I/O.
- Use `pgxpool.Pool` for all database access; never open a raw `sql.DB` for PostgreSQL.
- `internal/` packages are not imported by external packages — this is enforced by Go's module system.
- Format with `gofmt` before committing.

### TypeScript / React

- No `any` types. Use explicit types or `unknown` with narrowing.
- All API calls go through `src/lib/api.ts` — no direct `fetch()` calls in components.
- All visual styling must follow `DESIGN.md`: token names, typography, spacing, radius, component states, page layouts, and Material Symbols icon names.
- TanStack Query for all server state. No `useEffect` for data fetching.
- Zustand for client UI state only (auth token, sidebar open, theme).
- Components are function components. No class components.
- All pages have loading and error states.
- Format with Prettier before committing.
- Use Material Symbols Outlined via Google Fonts; do not introduce lucide-react or unrelated icon packs.

### SQL

- All schema-modifying queries use DDL transactions where PostgreSQL supports them.
- `_rift_migrations` table bootstrap is idempotent: use `CREATE TABLE IF NOT EXISTS`.
- Advisory lock key is always `hashtext('rift_migration_lock')` — never hardcode an integer.

---

## File Structure Reference

```
rift/
├── cmd/rift/main.go
├── internal/
│   ├── api/
│   │   ├── server.go
│   │   ├── routes.go
│   │   └── handlers/
│   ├── cli/
│   │   ├── new.go
│   │   ├── up.go
│   │   ├── down.go
│   │   ├── status.go
│   │   ├── diff.go
│   │   ├── lint.go
│   │   └── server.go
│   ├── config/config.go
│   ├── db/db.go
│   ├── diff/
│   │   ├── introspect.go
│   │   ├── parse.go
│   │   └── diff.go
│   ├── embed/ui/           ← go:embed target (built React app)
│   ├── linter/linter.go
│   └── migration/
│       ├── checksum.go
│       ├── conflict.go
│       ├── loader.go
│       ├── runner.go
│       └── state.go
├── web/
│   ├── src/
│   │   ├── lib/api.ts
│   │   ├── stores/
│   │   ├── hooks/
│   │   ├── components/
│   │   └── pages/
│   ├── package.json
│   └── vite.config.ts
├── migrations/             ← user migration files live here
├── docker/Dockerfile
├── docker-compose.yml
├── Makefile
├── rift.yaml.example
├── go.mod
├── go.sum
└── README.md
```

---

## Key Implementation Details

### Advisory Lock

```go
func AcquireAdvisoryLock(ctx context.Context, pool *pgxpool.Pool) (func(), error) {
    conn, err := pool.Acquire(ctx)
    if err != nil {
        return nil, fmt.Errorf("acquiring connection for advisory lock: %w", err)
    }
    var acquired bool
    err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtext('rift_migration_lock'))").Scan(&acquired)
    if err != nil {
        conn.Release()
        return nil, fmt.Errorf("trying advisory lock: %w", err)
    }
    if !acquired {
        conn.Release()
        return nil, fmt.Errorf("another rift migration is already in progress — try again shortly")
    }
    release := func() {
        conn.Exec(ctx, "SELECT pg_advisory_unlock(hashtext('rift_migration_lock'))")
        conn.Release()
    }
    return release, nil
}
```

### go:embed Directive

In `internal/api/server.go`:
```go
//go:embed ../../internal/embed/ui
var embeddedUI embed.FS

func serveUI(mux *chi.Mux) {
    sub, _ := fs.Sub(embeddedUI, "internal/embed/ui")
    mux.Handle("/*", http.FileServer(http.FS(sub)))
}
```

SPA fallback — all non-asset routes must return `index.html`:
```go
mux.Get("/*", func(w http.ResponseWriter, r *http.Request) {
    if strings.HasPrefix(r.URL.Path, "/api/") {
        http.NotFound(w, r)
        return
    }
    indexHTML, _ := sub.Open("index.html")
    defer indexHTML.Close()
    io.Copy(w, indexHTML)
})
```

### SSE for Live Deploy

```go
func HandleMigrateUp(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    events := make(chan string)
    go func() {
        // RunUp sends progress to events channel
        runner.RunUpWithEvents(r.Context(), pool, cfg, events)
        close(events)
    }()

    for msg := range events {
        fmt.Fprintf(w, "data: %s\n\n", msg)
        flusher.Flush()
    }
}
```

### pg_notify for Future Extension

In `runner.RunUp`, after each successful migration:
```go
conn.Exec(ctx, "SELECT pg_notify('rift_migration_applied', $1)", version)
```
This is wired but not consumed in v1 — sets up future WebSocket subscription.

### Schema Introspection Queries

Tables and columns:
```sql
SELECT
    t.table_name,
    c.column_name,
    c.data_type,
    c.is_nullable,
    c.column_default,
    c.character_maximum_length
FROM information_schema.tables t
JOIN information_schema.columns c ON c.table_name = t.table_name AND c.table_schema = t.table_schema
WHERE t.table_schema = $1
  AND t.table_type = 'BASE TABLE'
ORDER BY t.table_name, c.ordinal_position;
```

Indexes:
```sql
SELECT
    indexname,
    tablename,
    indexdef
FROM pg_indexes
WHERE schemaname = $1
ORDER BY tablename, indexname;
```

Foreign keys:
```sql
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    tc.constraint_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = $1;
```

---

## Linter Pattern Reference

Implement each pattern as a regex scan over the SQL string (case-insensitive):

| ID | Regex Pattern | Severity | Message |
|---|---|---|---|
| L001 | `DROP\s+COLUMN` | error | "Dropping a column is irreversible. Consider renaming to `_deprecated_*` first." |
| L002 | `RENAME\s+COLUMN` | error | "Renaming a column breaks running applications. Add a new column, migrate data, then drop the old." |
| L003 | `SET\s+NOT\s+NULL` (without DEFAULT in same statement) | error | "Adding NOT NULL without a DEFAULT causes table lock and fails on existing rows. Set DEFAULT first." |
| L004 | `DROP\s+TABLE` | error | "Dropping a table is irreversible. Rename to `_deprecated_*` and archive data before dropping." |
| L005 | `CREATE\s+INDEX(?!\s+CONCURRENTLY)` | warning | "CREATE INDEX locks the table. Use `CREATE INDEX CONCURRENTLY` to avoid downtime." |
| L006 | `ADD\s+CONSTRAINT.*FOREIGN\s+KEY` (without NOT VALID) | warning | "Foreign key constraints cause a full table scan. Add with `NOT VALID`, then `VALIDATE CONSTRAINT` separately." |

---

## React Page Specifications

`DESIGN.md` is authoritative for every visual decision. The route list below is a functional implementation summary only; when class names, layout, icons, colors, or component states differ, implement `DESIGN.md`.

### Global App Shell
- Fixed 240px sidebar on desktop using the exact navigation structure from `DESIGN.md` Section 12: Dashboard (`/`), Migrations (`/migrations`), Schema Diff (`/migrations/:version/diff`), Team (`/team`), Settings (`/settings`), plus Docs and Support footer links.
- Sticky top app bar with Docs/API/Changelog links, `Preview Changes`, `Apply Migrations`, notification icon, and account icon.
- Tailwind tokens, fonts, border radii, component spacing, and Material Symbols icons come from `DESIGN.md` Sections 2–11.

### `/` and `/migrations` — Migration Dashboard
- `/` should redirect to or render the same dashboard as `/migrations`.
- Three stat cards: Total Migrations, Applied, Pending. Pending uses the elevated primary glow treatment when pending count > 0.
- Recent Activity table columns: Status | ID (Timestamp) | Name | Author | Applied Date | Actions.
- Right sidebar: Quick Actions card (`Connect to DB`, `Sync Local Files`) and Linter Alerts card showing dangerous pending migrations.
- Search input filters migrations by name or version.

### `/migrations/:version` — SQL Authoring Interface
- Three-panel layout: left schema/table browser, center CodeMirror SQL editor, right Zero-Downtime Linter panel.
- Metadata strip above editor: editable filename, category dropdown, author badge.
- Linter issues render as red left-accent cards with Auto-Fix buttons; passing rules render with secondary green dots.

### `/migrations/:version/diff` — Schema Diff Viewer
- Custom full-height split-pane viewer, not a generic default diff page: left `LOCAL MIGRATIONS`, right `LIVE DATABASE`.
- Summary/control panel above panes with change counts and `SAFE PREVIEW (DDL)` toggle.
- Added lines use secondary emerald treatment; deleted lines use error treatment with strikethrough.
- `APPLY MIGRATIONS` opens the confirmation modal and triggers `POST /api/v1/migrate/up` with SSE feedback surfaced inline.

### `/team` — Team & Deployment History
- Two-column layout: Conflict Detection and Deployment History timeline on the left; Team Access and Notifications/Webhooks panels on the right.
- Deployment History consumes `GET /api/v1/history`; conflicts consume `GET /api/v1/conflicts`; team members consume `GET /api/v1/team`.
- Slack/Discord webhook UI is configuration-only in v1; do not implement delivery logic.

### `/settings` — Settings
- Token field (masked, reveal toggle), environment name, and database connection status indicator.
- Save button persists token/UI settings to `localStorage` and Zustand.

---

## Definition of Done

The project is complete when:

- [ ] All 10 phases are committed and pushed
- [ ] `go test ./...` passes with 0 failures
- [ ] `cd web && npm run test` passes with 0 failures
- [ ] `make build` produces a single `./rift` binary with embedded UI
- [ ] `docker compose up` brings up Rift + Postgres and serves the dashboard
- [ ] Frontend routes match the `DESIGN.md` mockup-backed layouts and token system
- [ ] `rift up`, `rift down`, `rift diff`, `rift status`, `rift lint` all work correctly against a real PostgreSQL database
- [ ] The schema diff viewer correctly identifies and displays changes for at least: CREATE TABLE, ALTER TABLE ADD COLUMN, ALTER TABLE DROP COLUMN, CREATE INDEX
- [ ] The linter correctly flags all 6 dangerous patterns
- [ ] README has a working 5-minute quickstart

---

## What NOT To Do

- Do not use `database/sql` with the standard `lib/pq` driver for PostgreSQL — use `pgx/v5` exclusively.
- Do not use `any` types in TypeScript — use explicit types or `unknown` with narrowing.
- Do not use class components in React.
- Do not use `useEffect` for data fetching — use TanStack Query.
- Do not commit `node_modules/`, build artifacts, or secrets.
- Do not skip the advisory lock when running migrations.
- Do not use `os.Exit()` in library code — only in `cmd/rift/main.go`.
- Do not hardcode database credentials anywhere in source code.
- Do not implement RBAC, OAuth, or multi-user auth in v1.
- Do not add Elasticsearch integration in v1.
- Do not proceed to the next phase if any verification check fails.