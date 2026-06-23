# Rift

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=111)
![License](https://img.shields.io/badge/license-MIT-green)

Rift is a self-hosted PostgreSQL migration manager: **Flyway-style SQL migrations plus a React dashboard for schema diffs, linter warnings, deployment history, and team visibility.**

## Why Rift?

Database migrations are risky because most tools apply SQL without showing what will change in the live schema. Rift keeps the normal developer workflow — timestamped `*.up.sql` / `*.down.sql` files — and adds the missing safety rails:

- **Schema diff preview** before apply: local migration intent vs live PostgreSQL state.
- **Zero-downtime linter** for dangerous DDL patterns such as `DROP COLUMN`, unsafe `CREATE INDEX`, and foreign keys without `NOT VALID`.
- **Team conflict detection** using checksums stored in `_rift_migrations`.
- **Audit history** with author, execution time, rollback state, and deployment timeline.
- **Single binary deployment**: Go API + embedded React dashboard.
- **Docker Compose quickstart** for local evaluation.

## Screenshots

| Dashboard | Schema Diff |
|---|---|
| ![Rift migration dashboard](doc/rift_dashboard.png) | ![Rift schema diff viewer](doc/schema_diff_viewer.png) |

| SQL Authoring | Team & Deployment |
|---|---|
| ![Rift SQL authoring interface](doc/sql_authoring_interface.png) | ![Rift team deployment history](doc/team_deployment_history.png) |

## 5-minute quickstart

### Prerequisites

- Docker Engine + Docker Compose
- Port `7878` available for Rift
- Port `15432` available for the bundled PostgreSQL instance

### Start Rift

```bash
git clone git@github.com:fandykun/rift.git
cd rift
docker compose up -d --build --wait
```

Open the dashboard:

```text
http://localhost:7878
```

Use the local development API token when prompted:

```text
local-dev-token
```

Check the API directly:

```bash
RIFT_TOKEN=local-dev-token
AUTH_SCHEME=Bearer
curl --header "Authorization: ${AUTH_SCHEME} ${RIFT_TOKEN}" http://localhost:7878/api/v1/status
```

Stop the stack:

```bash
docker compose down
```

## Demo deployment with dummy data

Use the demo Compose overlay when you want a populated Rift dashboard instead of an empty first-run database:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo.yml up -d --build --wait
docker compose -f docker-compose.yml -f docker-compose.demo.yml --profile seed run --rm demo-seed
```

The overlay mounts `demo/migrations` into the Rift container. The one-shot `demo-seed` service applies three sample migrations:

- `demo_customers` — customer accounts across free, team, and enterprise plans.
- `demo_projects` — projects linked to customers with a `NOT VALID` foreign key.
- `demo_deployment_audit` — example deployment audit rows.

Verify the seeded API state:

```bash
RIFT_TOKEN=local-dev-token
AUTH_SCHEME=Bearer
curl --header "Authorization: ${AUTH_SCHEME} ${RIFT_TOKEN}" http://localhost:7878/api/v1/status
```

Expected counts after the seed service completes:

```json
{"environment":"demo","counts":{"applied":3,"pending":0,"rolled_back":0,"total":3}}
```

Stop the demo stack:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo.yml down
```

## CLI reference

Rift reads configuration from `rift.yaml`, with environment variable overrides such as `RIFT_DATABASE_URL`, `RIFT_MIGRATIONS_DIR`, `RIFT_PORT`, and `RIFT_TOKEN`.

| Command | Purpose |
|---|---|
| `rift new <name>` | Create timestamped `*.up.sql` and `*.down.sql` migration files. |
| `rift status` | Show applied, pending, and rolled-back migrations. |
| `rift lint [file]` | Scan pending migrations or one file for dangerous DDL. |
| `rift diff` | Compare migration-derived schema against the live database. |
| `rift up [--dry-run] [--force]` | Apply pending migrations with advisory locking and linter checks. |
| `rift down [--steps N]` | Roll back the latest applied migrations after confirmation. |
| `rift config check` | Validate config loading and database connectivity. |
| `rift server` | Start the API server and embedded dashboard. |
| `rift --version` | Print the version and build commit. |

## Configuration

Example `rift.yaml`:

```yaml
environment: development
database_url: postgres://rift:rift_dev_password@localhost:15432/rift?sslmode=disable
migrations_dir: ./migrations
author: fandy@example.com

server:
  port: 7878
  token: local-dev-token

linter:
  warn_only: false

team:
  - name: Fandy
    email: fandy@example.com
    role: admin
```

Environment variables override the file:

| Variable | Purpose |
|---|---|
| `RIFT_ENV` | Environment label shown in API/dashboard. |
| `RIFT_DATABASE_URL` | PostgreSQL connection URL. |
| `RIFT_MIGRATIONS_DIR` | Directory containing migration files. |
| `RIFT_PORT` | HTTP server port. |
| `RIFT_TOKEN` | Bearer token required for API requests. |

## Architecture

```text
┌────────────────────────────┐
│        React Dashboard      │
│  migrations · diff · team   │
└──────────────┬─────────────┘
               │ embedded assets / REST + SSE
┌──────────────▼─────────────┐
│          Go API             │
│ Chi routes · auth · CORS    │
└───────┬───────────┬────────┘
        │           │
        │           └────────────────────┐
        │                                │
┌───────▼────────┐             ┌─────────▼─────────┐
│ Migration Core │             │ Schema Diff Engine │
│ loader/runner  │             │ parse/introspect   │
│ advisory locks │             │ render JSON/CLI    │
└───────┬────────┘             └─────────┬─────────┘
        │                                │
        └──────────────┬─────────────────┘
                       │
             ┌─────────▼─────────┐
             │    PostgreSQL      │
             │ _rift_migrations   │
             │ information_schema │
             └───────────────────┘
```

## Development

Install frontend dependencies once:

```bash
cd web
npm install
```

Run verification:

```bash
go test ./...
cd web && npm run test
cd web && npm run build
```

Build a single binary with embedded UI:

```bash
make build
./rift --version
```

Run locally without Docker:

```bash
RIFT_DATABASE_URL='postgres://rift:rift_dev_password@localhost:15432/rift?sslmode=disable' \
RIFT_TOKEN='local-dev-token' \
./rift server
```

## Linter rules

| Pattern | Severity | Safer path |
|---|---:|---|
| `DROP COLUMN` | Error | Deprecate first, migrate reads/writes, then drop later. |
| `RENAME COLUMN` | Error | Add new column, backfill, dual-write, then remove old column. |
| `ALTER COLUMN ... SET NOT NULL` without default/backfill | Error | Set default, backfill, then enforce constraint. |
| `DROP TABLE` | Error | Rename/archive first. |
| `CREATE INDEX` without `CONCURRENTLY` | Warning | Use `CREATE INDEX CONCURRENTLY` where possible. |
| `ADD CONSTRAINT ... FOREIGN KEY` without `NOT VALID` | Warning | Add `NOT VALID`, then validate separately. |

## Repository layout

```text
cmd/rift/             Cobra CLI entrypoint
internal/api/         Chi API server, REST handlers, SSE migration stream
internal/migration/   migration loader, state table, runner, advisory lock
internal/diff/        schema introspection, SQL parsing, diff rendering
internal/linter/      dangerous DDL scanner
internal/embed/ui/    embedded dashboard build output
web/                  React + TypeScript dashboard
docker/               Dockerfile for single-binary runtime
```

## Status

Rift is currently implemented through Phase 10 of `BUILD_PHASES.md`:

- CLI migration workflow
- API server
- React dashboard
- SQL authoring page
- schema diff viewer
- team/deployment page
- settings page
- single binary embed
- Docker Compose deployment
- React/Go tests
- README quickstart
- CLI progress/version polish
- UI empty states and theme toggle

## Contributing

1. Create a branch from `main`.
2. Keep changes small and independently revertible.
3. Run `go test ./...`, `cd web && npm run test`, and `cd web && npm run build` before submitting.
4. Include screenshots for dashboard-facing UI changes.
5. Document migration behavior changes in this README and `BUILD_PHASES.md` when relevant.
