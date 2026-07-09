-- 0003_diary.up.sql
-- Diary feature (second vertical slice). A user's diary entries, designed for
-- offline-first clients: the client generates client_id (a UUID) so an entry
-- created offline can be POSTed idempotently once connectivity returns, without
-- ever creating a duplicate. See docs/adding-a-feature.md and apps/go/README.md.

CREATE TABLE IF NOT EXISTS diary_entries (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body_text  TEXT        NOT NULL DEFAULT '',
    -- mood is optional (nullable). It is saved now but only consumed by a later
    -- insights slice; the diary feature itself never reads it.
    mood       TEXT,
    -- client_id is the client-generated UUID. The unique index below turns POST
    -- into an idempotent upsert keyed by (user_id, client_id).
    client_id  UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Soft delete: deletion sets deleted_at so the /diary/sync delta can carry the
    -- tombstone to other devices. The list endpoint filters these out.
    deleted_at TIMESTAMPTZ
);

-- Idempotency + duplicate prevention: one entry per (user, client_id). A retried
-- offline create upserts onto the same row instead of inserting a second copy.
CREATE UNIQUE INDEX IF NOT EXISTS diary_entries_user_client_key
    ON diary_entries (user_id, client_id);

-- Keyset pagination for GET /diary (newest first): ORDER BY created_at DESC, id
-- DESC with a (created_at, id) cursor rides this index.
CREATE INDEX IF NOT EXISTS diary_entries_user_created_idx
    ON diary_entries (user_id, created_at DESC, id DESC);

-- Delta sync for GET /diary/sync?since=: rows changed after a timestamp.
CREATE INDEX IF NOT EXISTS diary_entries_user_updated_idx
    ON diary_entries (user_id, updated_at);
