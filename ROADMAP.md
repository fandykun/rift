# Roadmap

Rift is being built in public milestones. This roadmap summarizes what is already working and what is planned next without exposing internal task-management detail.

## Current Status

Rift has a working MVP for self-hosted PostgreSQL migration management:

- CLI migration workflow for creating, applying, rolling back, linting, and diffing SQL migrations
- PostgreSQL-backed migration state with checksums, rollback markers, and advisory locking
- Schema diff engine that compares migration intent against live PostgreSQL state
- Dangerous DDL linter for common zero-downtime migration risks
- React dashboard with migration list, SQL authoring, schema diff viewer, team/deployment page, and settings
- Authenticated REST API and SSE migration apply stream
- Single-binary Go deployment with embedded React UI
- Docker Compose quickstart and demo deployment support
- Public deployment readiness checks via `rift config doctor`

## Completed Milestones

### Foundation

- Go CLI scaffold with Cobra
- React/TypeScript dashboard scaffold
- Shared configuration loading from `rift.yaml` and environment variables
- Docker and Makefile build targets

### Migration Engine

- Timestamped `*.up.sql` / `*.down.sql` migration file loading
- `_rift_migrations` state table
- Advisory locks to prevent concurrent migration runs
- Apply and rollback workflow
- Conflict detection for missing files and checksum mismatches

### Safety Tooling

- Schema introspection from PostgreSQL catalogs
- Migration SQL parsing for common DDL operations
- Structured schema diff output for CLI and API use
- Linter warnings for risky DDL patterns such as `DROP COLUMN`, unsafe `CREATE INDEX`, and foreign keys without `NOT VALID`

### Web Dashboard

- Migration dashboard with status cards, search, linter alerts, and action links
- SQL authoring route for migration details
- Schema diff viewer with local-vs-live panes
- Team/deployment page with conflicts, history, and team access
- Settings page with token persistence and connection status
- New Migration modal that creates timestamped migration file pairs through the authenticated API
- SQL editor persistence for pending `up.sql` and `down.sql` files
- Confirmed rollback of the latest applied migration from the dashboard, with post-rollback data refresh

### Deployment

- Embedded React UI in the Go binary
- Docker Compose local quickstart
- Demo migrations and seed flow
- Public deployment guide
- VPS deployment support with persistent host-mounted migration storage

## In Progress

- Polish migration creation templates for common PostgreSQL changes
- Improve public demo screenshots and documentation
- Tighten deployment examples for different hosting targets

## Planned

### Near Term

- Richer migration authoring templates
- Better empty/error states across all dashboard routes
- More complete schema diff coverage for indexes and constraints
- Safer apply confirmation flow with clearer linter gating

### Later

- GitHub Actions / CI integration examples
- Slack/Discord webhook delivery for deployment events
- Role-based access control or OAuth for team deployments
- Multi-database exploration after PostgreSQL support is mature

## Release Criteria

Before tagging a public release, Rift should consistently pass:

```bash
go test ./...
cd web && npm run test
cd web && npm run lint
cd web && npm run build
make build
```

The Docker quickstart and public deployment guide should also be verified from a clean checkout.
