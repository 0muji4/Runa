-- One curated quote per calendar day; the admin upsert is ON CONFLICT (date).
CREATE TABLE IF NOT EXISTS daily_quotes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    date       DATE        NOT NULL UNIQUE,
    body_text  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One curated song per calendar day.
CREATE TABLE IF NOT EXISTS daily_songs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    date        DATE        NOT NULL UNIQUE,
    title       TEXT        NOT NULL,
    artist      TEXT        NOT NULL,
    artwork_url TEXT        NOT NULL,
    audio_url   TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Keyset pagination with a (date, id) cursor.
CREATE INDEX IF NOT EXISTS daily_songs_date_idx
    ON daily_songs (date DESC, id DESC);

-- Append-only play log, one row per play.
CREATE TABLE IF NOT EXISTS song_history (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID        NOT NULL REFERENCES users (id)       ON DELETE CASCADE,
    song_id   UUID        NOT NULL REFERENCES daily_songs (id) ON DELETE CASCADE,
    played_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS song_history_user_played_idx
    ON song_history (user_id, played_at DESC);
