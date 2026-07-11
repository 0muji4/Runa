-- 0004_today.down.sql
-- Reverts 0004_today.up.sql. Dropping each table removes its indexes too.
-- song_history references daily_songs, so it is dropped first.

DROP TABLE IF EXISTS song_history;
DROP TABLE IF EXISTS daily_songs;
DROP TABLE IF EXISTS daily_quotes;
