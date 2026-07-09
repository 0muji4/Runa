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

| Method | Path                    | Response                                   |
| ------ | ----------------------- | ------------------------------------------ |
| GET    | `/api/v1/healthz`       | `200 {"status":"ok"}`                      |
| POST   | `/api/v1/auth/signup`   | `201 AuthTokens` — email + password        |
| POST   | `/api/v1/auth/login`    | `200 AuthTokens` — email + password        |
| POST   | `/api/v1/auth/apple`    | `200 AuthTokens` — Apple ID token          |
| POST   | `/api/v1/auth/google`   | `200 AuthTokens` — Google ID token         |
| POST   | `/api/v1/auth/refresh`  | `200 AuthTokens` — rotates the refresh token |
| POST   | `/api/v1/auth/logout`   | `204` — revoke a refresh token (idempotent) |
| GET    | `/api/v1/me`            | `200 User` — **requires `Authorization: Bearer`** |
| GET    | `/api/v1/diary`         | `200 DiaryListResponse` — newest first, keyset paging (`limit`, `cursor`) |
| POST   | `/api/v1/diary`         | `201`/`200 DiaryEntry` — idempotent by `client_id`   |
| GET    | `/api/v1/diary/{id}`    | `200 DiaryEntry` (`404` if not the caller's)         |
| PATCH  | `/api/v1/diary/{id}`    | `200 DiaryEntry` — replaces `body_text` + `mood`     |
| DELETE | `/api/v1/diary/{id}`    | `204` — soft delete (idempotent)                     |
| GET    | `/api/v1/diary/sync`    | `200 DiarySyncResponse` — changes since `?since=` (incl. tombstones) |

All `/api/v1/diary*` routes **require `Authorization: Bearer`** and only ever
touch the caller's own entries. The full contract lives in
[`api/openapi.yaml`](api/openapi.yaml) and grows with each new endpoint.

### Auth design

- **Access token**: short-lived HS256 JWT (`sub`=user id). **Refresh token**:
  long-lived opaque 256-bit value; only its SHA-256 hash is stored, and it is
  rotated (old revoked, new issued) on every `/auth/refresh`.
- **Passwords**: hashed with **Argon2id** (OWASP parameters), stored as a PHC
  string. **Apple/Google**: the ID token signature is verified against each
  provider's JWKS, checking `iss` / `aud` (from `APPLE_CLIENT_IDS` /
  `GOOGLE_CLIENT_IDS`) / `exp`.
- **Errors**: one envelope for all endpoints — `{"error":{"code","message","details?}}`.
  `login` / `signup` are rate limited per client IP.
- Crypto/token primitives live in `internal/auth/`; the flow is
  `internal/repository/auth_repository.go` → `internal/service/auth.go` →
  `internal/handler/auth.go`, mounted in `internal/server/router.go`.

### Diary design (offline-first sync)

The diary is built for clients that must write while offline and reconcile on
reconnect. Two contract choices make that safe:

- **`client_id` idempotency.** The client generates `client_id` (a UUID) when an
  entry is authored, and it is unique per `(user_id, client_id)`. `POST /diary`
  upserts on that key, so retrying a create that was queued offline never makes a
  duplicate — the first insert returns `201`, a repeat returns `200` with the same
  row. `created_at` is client-supplied so an offline entry keeps its authored time.
- **Delta sync with tombstones + last-write-wins.** `GET /diary/sync?since=`
  returns every row whose `updated_at` is after the watermark, **including
  soft-deleted rows** (`deleted_at` set) so other devices learn of deletions. The
  response's `server_time` becomes the client's next `since` (`last_synced_at`).
  Conflicts are resolved last-write-wins by `updated_at`; there is no server-side
  merge. Ownership is enforced in the data layer — a query for another user's row
  returns not-found, and the handler answers `404` so existence never leaks.
- The flow mirrors auth: `internal/repository/diary_repository.go` →
  `internal/service/diary.go` → `internal/handler/diary.go`.

## Configuration

All configuration is read from environment variables (see `.env.example`).

| Variable               | Default                                                        | Description                                            |
| ---------------------- | -------------------------------------------------------------- | ------------------------------------------------------ |
| `PORT`                 | `8080`                                                         | TCP port the HTTP server listens on.                   |
| `DATABASE_URL`         | `postgres://runa:runa@localhost:5432/runa?sslmode=disable`     | Postgres connection string (pgx/libpq format).         |
| `LOG_LEVEL`            | `info`                                                         | slog level: `debug` \| `info` \| `warn` \| `error`.    |
| `CORS_ALLOWED_ORIGINS` | `*`                                                           | Comma-separated allowed CORS origins.                  |
| `APP_ENV`              | `development`                                                  | Deployment environment.                                |
| `JWT_SECRET`           | `dev-insecure-secret-change-me`                               | HS256 secret for access tokens. **Override in prod.**  |
| `ACCESS_TOKEN_TTL`     | `15m`                                                         | Access-token lifetime (Go duration).                   |
| `REFRESH_TOKEN_TTL`    | `720h`                                                        | Refresh-token lifetime (30 days).                      |
| `APPLE_CLIENT_IDS`     | (empty)                                                       | Accepted `aud` for Apple ID tokens (Bundle ID / Service ID, comma-separated). |
| `GOOGLE_CLIENT_IDS`    | (empty)                                                       | Accepted `aud` for Google ID tokens (OAuth client IDs, comma-separated). |

Getting the provider audiences:

- **Google**: from Google Cloud → APIs & Services → Credentials, create OAuth
  client IDs for iOS, Android (needs the app's SHA-1) and Web/"server". Put the
  ones whose tokens reach the backend (typically the iOS, Android and Web client
  IDs) into `GOOGLE_CLIENT_IDS`.
- **Apple**: from the Apple Developer portal, the iOS app **Bundle ID**
  (native Sign in with Apple) and/or a **Service ID** (Android web flow). Put
  them into `APPLE_CLIENT_IDS`.

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
enables the `pgcrypto` extension for `gen_random_uuid()`. `0002_auth` extends
`users` with the identity/credential columns (email, auth_provider, apple_sub,
google_sub, display_name, password_hash, is_premium, premium_expires_at) and adds
the `refresh_tokens` table. `0003_diary` adds `diary_entries` (body/mood/client_id
+ timestamps + soft-delete `deleted_at`), with a unique `(user_id, client_id)`
index for idempotent upsert and indexes for keyset paging and delta sync.

## Tests

```sh
go test ./...   # unit + auth-flow integration; no database required (in-memory store)
```

Auth tests cover the argon2id/JWT/refresh/OIDC primitives (`internal/auth`), the
service and handler, and a full `signup → /me → refresh → logout` flow through the
real router (`internal/server/auth_flow_test.go`). Diary tests cover the service
(idempotent upsert, ownership, keyset paging, soft-delete/sync delta), the handler
(validation, 201-vs-200, 404), and a full `create → list → get → patch → sync →
delete` flow with a real Bearer token (`internal/server/diary_flow_test.go`),
including cross-user `404`. CI has no Postgres, so the tests use the in-memory
`internal/repository/memauth` and `internal/repository/memdiary` stores; the
pgx-backed repositories are exercised by running the server against
`docker compose`.

## Versions

Pinned starting point (may need local alignment):

- Go 1.23
- Postgres 16
- chi v5, pgx v5, golang-migrate v4
