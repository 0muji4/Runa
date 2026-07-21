-- 0006_devices.up.sql
-- Devices feature (eighth vertical slice). Registers a client's push token so the
-- backend can, in a FUTURE slice, send server-initiated notifications (FCM/APNs).
-- This slice's nightly reminder is a LOCAL notification scheduled on-device, so
-- this table is the registration "口" only — nothing here sends a push yet. See
-- docs/adding-a-feature.md and apps/go/README.md.

-- One row per (user, push_token). push_token is the device's FCM/APNs token;
-- platform disambiguates the delivery channel; notify_time (local "HH:MM") and
-- enabled mirror the user's reminder preference so a future server-side sender
-- knows when/whether to push. A repeated PUT upserts onto the same row.
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

-- Idempotency + duplicate prevention: one row per (user, push_token). A repeated
-- PUT /devices upserts onto the same row instead of inserting a second copy.
CREATE UNIQUE INDEX IF NOT EXISTS devices_user_token_key
    ON devices (user_id, push_token);
