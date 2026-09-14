-- push_token is the device's FCM/APNs token; notify_time is the local "HH:MM"
-- reminder preference. Nothing server-side sends a push yet.
CREATE TABLE IF NOT EXISTS devices (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    push_token  TEXT        NOT NULL,
    platform    TEXT        NOT NULL CHECK (platform IN ('ios', 'android')),
    notify_time TEXT        NOT NULL,
    enabled     BOOLEAN     NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One row per (user, push_token): a repeated PUT upserts onto the same row.
CREATE UNIQUE INDEX IF NOT EXISTS devices_user_token_key
    ON devices (user_id, push_token);
