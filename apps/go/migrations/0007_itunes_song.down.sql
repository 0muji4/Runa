-- 0007_itunes_song.down.sql
-- Rows deleted by 0007 cannot be restored; only the columns return to the 0004
-- shape.
DELETE FROM daily_songs;

ALTER TABLE daily_songs
    DROP COLUMN IF EXISTS itunes_track_id,
    DROP COLUMN IF EXISTS store_url,
    DROP COLUMN IF EXISTS resolved_at;

ALTER TABLE daily_songs RENAME COLUMN preview_url TO audio_url;
