-- Rows that lived without a mood get the former client default.
ALTER TABLE gallery_images
    ADD COLUMN IF NOT EXISTS theme TEXT NOT NULL DEFAULT 'pink' CHECK (theme IN ('monotone', 'pink'));
ALTER TABLE gallery_images ALTER COLUMN theme DROP DEFAULT;
