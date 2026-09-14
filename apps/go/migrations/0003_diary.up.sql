CREATE TABLE IF NOT EXISTS diary_entries (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body_text  TEXT        NOT NULL DEFAULT '',
    mood       TEXT,
    -- client_id is client-generated so an offline create can be retried idempotently.
    client_id  UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Soft delete: the tombstone is carried to other devices by /diary/sync.
    deleted_at TIMESTAMPTZ
);

-- One entry per (user, client_id): a retried offline create upserts onto the same row.
CREATE UNIQUE INDEX IF NOT EXISTS diary_entries_user_client_key
    ON diary_entries (user_id, client_id);

-- Keyset pagination with a (created_at, id) cursor.
CREATE INDEX IF NOT EXISTS diary_entries_user_created_idx
    ON diary_entries (user_id, created_at DESC, id DESC);

-- Delta sync: rows changed after a timestamp.
CREATE INDEX IF NOT EXISTS diary_entries_user_updated_idx
    ON diary_entries (user_id, updated_at);
