-- itunes_track_id becomes the source of truth; the other columns hold metadata
-- fetched from the iTunes Search API at resolved_at.
-- Existing rows carry no Apple identifier, so every row (and its song_history,
-- via ON DELETE CASCADE) is removed; the operator re-registers songs by track id.
DELETE FROM daily_songs;

ALTER TABLE daily_songs RENAME COLUMN audio_url TO preview_url;

ALTER TABLE daily_songs
    ADD COLUMN itunes_track_id BIGINT      NOT NULL,
    ADD COLUMN store_url       TEXT        NOT NULL,
    ADD COLUMN resolved_at     TIMESTAMPTZ NOT NULL;
