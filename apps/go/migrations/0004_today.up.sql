-- 0004_today.up.sql
-- Today feature (third vertical slice). Powers the home screen's three daily
-- elements: a curated poetic quote and a curated song per day, plus a per-user
-- play history. The moon phase (the home's third element) is computed on the
-- client in shared code, so it has no table here. See docs/adding-a-feature.md.

-- One curated quote per calendar day. `date` is UNIQUE so GET /today can look a
-- day up directly and the admin upsert (ON CONFLICT (date)) replaces a day's copy.
CREATE TABLE IF NOT EXISTS daily_quotes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    date       DATE        NOT NULL UNIQUE,
    body_text  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One curated song per calendar day. artwork_url/audio_url are the player's
-- image and stream sources; both are required for a playable entry.
CREATE TABLE IF NOT EXISTS daily_songs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    date        DATE        NOT NULL UNIQUE,
    title       TEXT        NOT NULL,
    artist      TEXT        NOT NULL,
    artwork_url TEXT        NOT NULL,
    audio_url   TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Archive keyset pagination for GET /songs (newest first): ORDER BY date DESC,
-- id DESC with a (date, id) cursor rides this index.
CREATE INDEX IF NOT EXISTS daily_songs_date_idx
    ON daily_songs (date DESC, id DESC);

-- Append-only play log. One row per play; recorded by POST /songs/{id}/played.
CREATE TABLE IF NOT EXISTS song_history (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID        NOT NULL REFERENCES users (id)       ON DELETE CASCADE,
    song_id   UUID        NOT NULL REFERENCES daily_songs (id) ON DELETE CASCADE,
    played_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A user's recent plays, newest first.
CREATE INDEX IF NOT EXISTS song_history_user_played_idx
    ON song_history (user_id, played_at DESC);
