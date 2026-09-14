-- Image metadata only; the bytes live in object storage at object_key
-- ("gallery/{user_id}/{uuid}"). theme is the per-image color mood chosen at
-- upload, not the client's display theme. deleted_at is a soft delete.
CREATE TABLE IF NOT EXISTS gallery_images (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    object_key TEXT        NOT NULL,
    width      INTEGER     NOT NULL,
    height     INTEGER     NOT NULL,
    theme      TEXT        NOT NULL CHECK (theme IN ('monotone', 'pink')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- object_key is globally unique and is the idempotency key of a retried registration.
CREATE UNIQUE INDEX IF NOT EXISTS gallery_images_object_key_idx
    ON gallery_images (object_key);

-- Keyset pagination with a (created_at, id) cursor; partial so soft-deleted rows never widen the scan.
CREATE INDEX IF NOT EXISTS gallery_images_user_created_idx
    ON gallery_images (user_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
