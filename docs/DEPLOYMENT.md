# Rift Deployment Guide

This guide covers public deployment for the single-container Rift app. Rift serves the Go API and embedded React dashboard from one binary.

## Runtime requirements

- A PostgreSQL database reachable from the app container.
- A strong `RIFT_TOKEN` for dashboard/API access.
- A writable migrations directory for real deployments, or the baked demo migrations for public demos.

## Required environment variables

| Variable | Required | Description |
|---|---:|---|
| `RIFT_DATABASE_URL` | Yes | PostgreSQL URL. Inject this from the platform secret manager. |
| `RIFT_TOKEN` | Yes | Bearer token users enter in the dashboard. Use a generated secret, never `local-dev-token` for public URLs. |
| `RIFT_ENV` | No | Environment label shown in the UI. Use `production`, `staging`, or `demo`. |
| `RIFT_MIGRATIONS_DIR` | No | Defaults to `./migrations`. Use `/app/demo-migrations` for the public demo dataset. |
| `RIFT_PORT` | No | Explicit server port. |
| `PORT` | No | Platform-provided port fallback used when `RIFT_PORT` is unset. |

Generate a public-demo token locally:

```bash
openssl rand -hex 32
```

## Health check

Rift exposes an unauthenticated health endpoint:

```text
GET /healthz
```

Expected response:

```json
{"status":"ok","environment":"demo"}
```

The Docker image also defines a container health check against `/healthz`.

## Public demo deployment

Use this mode when you want a shareable Rift instance with dummy migration data.

Set platform variables:

```text
RIFT_ENV=demo
RIFT_MIGRATIONS_DIR=/app/demo-migrations
RIFT_DATABASE_URL=<managed-postgres-url>
RIFT_TOKEN=<generated-demo-token>
```

The Docker image includes the demo migration files at:

```text
/app/demo-migrations
```

After the app and database are deployed, run the seed command once as a one-off job or temporary release command:

```bash
./rift up
```

Expected API status after seeding:

```json
{"environment":"demo","counts":{"applied":3,"pending":0,"rolled_back":0,"total":3}}
```

Then open the public app URL and enter the configured `RIFT_TOKEN`.

## Railway example

1. Create a Railway project.
2. Add a managed PostgreSQL service.
3. Add a Rift service from this repository using `docker/Dockerfile`.
4. Set environment variables:

```text
RIFT_ENV=demo
RIFT_MIGRATIONS_DIR=/app/demo-migrations
RIFT_DATABASE_URL=${{Postgres.DATABASE_URL}}
RIFT_TOKEN=<generated-demo-token>
```

5. Deploy the Rift service.
6. Run a one-off seed command from the service shell or Railway job:

```bash
./rift up
```

7. Verify:

```bash
AUTH_SCHEME=Bearer
RIFT_TOKEN=<generated-demo-token>
curl https://<your-rift-domain>/healthz
curl --header "Authorization: ${AUTH_SCHEME} ${RIFT_TOKEN}" https://<your-rift-domain>/api/v1/status
```

## Render example

1. Create a PostgreSQL instance.
2. Create a Web Service from this repository using Docker.
3. Set environment variables:

```text
RIFT_ENV=demo
RIFT_MIGRATIONS_DIR=/app/demo-migrations
RIFT_DATABASE_URL=<Render internal database URL>
RIFT_TOKEN=<generated-demo-token>
```

4. Use `/healthz` as the health check path.
5. Run `./rift up` once from a shell/job to seed demo migrations.

## Production deployment mode

For a real database migration workflow, do **not** use `/app/demo-migrations`. Mount or bake your real migration files and set:

```text
RIFT_ENV=production
RIFT_MIGRATIONS_DIR=/app/migrations
RIFT_DATABASE_URL=<production-postgres-url>
RIFT_TOKEN=<strong-token>
```

Before exposing a production database, verify locally or in staging:

```bash
./rift config doctor
./rift config check
./rift lint
./rift status
./rift diff
```

Use `./rift config doctor --skip-db` only for build-time checks where the database is intentionally unavailable; public runtime validation should include the database connectivity check.

## Public access checklist

- [ ] `./rift config doctor` passes in the target environment.
- [ ] `RIFT_TOKEN` is a strong generated value.
- [ ] `RIFT_DATABASE_URL` comes from platform secrets, not git.
- [ ] `/healthz` returns `200` without auth.
- [ ] `/api/v1/status` returns `401` without auth.
- [ ] `/api/v1/status` returns `200` with the bearer token.
- [ ] Direct SPA routes such as `/settings` and `/team` return HTML.
- [ ] Demo seed was run exactly once, or is safely idempotent for the target database state.
