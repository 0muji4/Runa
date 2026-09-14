ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email              TEXT,
    ADD COLUMN IF NOT EXISTS auth_provider      TEXT        NOT NULL DEFAULT 'email',
    ADD COLUMN IF NOT EXISTS apple_sub          TEXT,
    ADD COLUMN IF NOT EXISTS google_sub         TEXT,
    ADD COLUMN IF NOT EXISTS display_name       TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS password_hash      TEXT,
    ADD COLUMN IF NOT EXISTS is_premium         BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS premium_expires_at TIMESTAMPTZ;

-- Partial unique indexes: accounts without an email / provider sub must not collide on NULLs.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key
    ON users (email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_apple_sub_key
    ON users (apple_sub) WHERE apple_sub IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_google_sub_key
    ON users (google_sub) WHERE google_sub IS NOT NULL;

-- token_hash is the SHA-256 of the token; the raw token is never stored.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);
