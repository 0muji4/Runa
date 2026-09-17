-- The gallery no longer grades photos, so the per-image color mood has nothing to describe.
ALTER TABLE gallery_images DROP COLUMN IF EXISTS theme;
