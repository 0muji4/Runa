-- 0007_itunes_song.up.sql
-- Today's song becomes a track from Apple's catalog (see
-- docs/dd/todays-song-itunes-preview.md). itunes_track_id is now the source of
-- truth; title/artist/artwork_url/preview_url/store_url hold the song metadata
-- fetched from the iTunes Search API, and resolved_at is when it was fetched.
-- Rows older than 24h are re-fetched in the background on read.
--
-- Existing rows carry no Apple identifier, so their metadata cannot be fetched:
-- every row is removed (song_history follows via ON DELETE CASCADE) and the
-- operator re-registers songs by track id.
DELETE FROM daily_songs;

ALTER TABLE daily_songs RENAME COLUMN audio_url TO preview_url;

ALTER TABLE daily_songs
    ADD COLUMN itunes_track_id BIGINT      NOT NULL,
    ADD COLUMN store_url       TEXT        NOT NULL,
    ADD COLUMN resolved_at     TIMESTAMPTZ NOT NULL;
