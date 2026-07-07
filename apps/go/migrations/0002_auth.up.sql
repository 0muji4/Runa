-- 0002_auth.up.sql
-- Auth feature (first vertical slice). Extends the users table with identity and
-- credential columns and adds refresh_tokens. See docs/adding-a-feature.md.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email              TEXT,
    ADD COLUMN IF NOT EXISTS auth_provider      TEXT        NOT NULL DEFAULT 'email',
    ADD COLUMN IF NOT EXISTS apple_sub          TEXT,
    ADD COLUMN IF NOT EXISTS google_sub         TEXT,
    ADD COLUMN IF NOT EXISTS display_name       TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS password_hash      TEXT,
    ADD COLUMN IF NOT EXISTS is_premium         BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS premium_expires_at TIMESTAMPTZ;

-- Uniqueness is enforced per-column and only when the value is present, so a
-- social-only account (no email) or an email-only account (no provider sub)
-- does not collide on NULLs.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key
    ON users (email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_apple_sub_key
    ON users (apple_sub) WHERE apple_sub IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_google_sub_key
    ON users (google_sub) WHERE google_sub IS NOT NULL;

-- Refresh tokens are stored as SHA-256 hashes only; a DB leak never exposes a
-- usable token. Rotation revokes the old row and inserts a new one.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);
