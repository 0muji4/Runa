-- 0005_gallery.up.sql
-- Gallery feature (sixth vertical slice). Stores only image METADATA; the image
-- bytes live in S3-compatible object storage and are exchanged directly between
-- the client and the store via short-lived presigned URLs (the backend issues
-- the URLs but never streams the bytes). See docs/adding-a-feature.md.

-- One row per uploaded image. object_key is the storage path
-- "gallery/{user_id}/{uuid}"; it is globally UNIQUE and doubles as the
-- idempotency key for POST /gallery (a retried registration upserts). theme is
-- the per-image "saved" color mood chosen at upload (monotone|pink) — distinct
-- from the gallery's client-side display-theme toggle. width/height let the grid
-- lay out masonry cells before the bytes load. deleted_at is a soft delete; the
-- object itself is removed asynchronously.
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

-- object_key is unique across all users (it embeds the user id) and backs the
-- ON CONFLICT upsert that makes registration idempotent.
CREATE UNIQUE INDEX IF NOT EXISTS gallery_images_object_key_idx
    ON gallery_images (object_key);

-- Keyset pagination for GET /gallery (newest first): ORDER BY created_at DESC,
-- id DESC with a (created_at, id) cursor rides this index. Partial on the live
-- rows so soft-deleted images never widen the scan.
CREATE INDEX IF NOT EXISTS gallery_images_user_created_idx
    ON gallery_images (user_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
