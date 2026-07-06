# Runa Backend

Go HTTP API for the Runa (moon-themed) app. This is the walking-skeleton
backend: it wires up configuration, logging, routing, Postgres, and migrations,
and serves a single health endpoint. No product features yet.

## Quick start

```sh
# From apps/go/
docker compose up --build

# In another terminal:
curl http://localhost:8080/api/v1/healthz
# -> {"status":"ok"}
```

To run the server directly (Go 1.23+) without Docker:

```sh
go run ./cmd/api
# The server still boots and serves /healthz even if Postgres is unreachable.
```

## API

Served under the base path `/api/v1`. Clients are injected host+port only and
append `/api/v1/...` themselves.

| Method | Path              | Response                    |
| ------ | ----------------- | --------------------------- |
| GET    | `/api/v1/healthz` | `200 {"status":"ok"}`       |

The full contract lives in [`api/openapi.yaml`](api/openapi.yaml) and grows with
each new endpoint.

## Configuration

All configuration is read from environment variables (see `.env.example`).

| Variable               | Default                                                        | Description                                            |
| ---------------------- | -------------------------------------------------------------- | ------------------------------------------------------ |
| `PORT`                 | `8080`                                                         | TCP port the HTTP server listens on.                   |
| `DATABASE_URL`         | `postgres://runa:runa@localhost:5432/runa?sslmode=disable`     | Postgres connection string (pgx/libpq format).         |
| `LOG_LEVEL`            | `info`                                                         | slog level: `debug` \| `info` \| `warn` \| `error`.    |
| `CORS_ALLOWED_ORIGINS` | `*`                                                           | Comma-separated allowed CORS origins.                  |
| `APP_ENV`              | `development`                                                  | Deployment environment.                                |

## Architecture

The backend is layered so each concern has a single home and dependencies point
inward (transport -> service -> repository). This keeps the health path trivial
today while giving feature code an obvious place to land.

```
cmd/api/main.go            Entry point: config, logger, DB pool, migrations,
                           router assembly, graceful shutdown.
internal/config            Env-based configuration loading.
internal/server            chi router + middleware chain (RequestID, RealIP,
                           Recoverer, structured request logger, CORS, Timeout).
internal/handler           HTTP transport: request/response <-> service. No logic.
internal/service           Application/business logic. Health service lives here.
internal/repository        Data access. Owns the pgx pool; repositories are stubs.
migrations                 golang-migrate SQL files (applied at startup).
api/openapi.yaml           The growing API contract.
```

Design notes:

- **Liveness is dependency-free.** `/api/v1/healthz` never touches the database,
  so it stays a pure liveness signal. The server intentionally keeps serving
  even when Postgres is unreachable at boot (it logs a warning and retries).
  Readiness checks (DB ping, downstream) will be a separate endpoint later.
- **Migrations run at startup** via golang-migrate `Up()` when the DB pool pings
  successfully; `ErrNoChange` is treated as success.

## Migrations

SQL migrations live in `migrations/` as `NNNN_name.up.sql` / `.down.sql` pairs.
They are applied automatically on startup when the database is reachable, and are
copied into the Docker image so the containerized server self-migrates.

`0001_init` creates a minimal `users` table stub (uuid pk + `created_at`) and
enables the `pgcrypto` extension for `gen_random_uuid()`.

## Versions

Pinned starting point (may need local alignment):

- Go 1.23
- Postgres 16
- chi v5, pgx v5, golang-migrate v4
