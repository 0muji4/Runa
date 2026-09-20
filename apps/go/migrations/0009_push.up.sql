-- No client has ever registered (the shared ApiClient had no /devices call), so
-- the table is emptied rather than backfilled before the NOT NULL columns land.
DELETE FROM devices;
DROP INDEX IF EXISTS devices_user_token_key;

ALTER TABLE devices
    -- install_id is generated once per app install and survives push-token rotation,
    -- so it (not the token) identifies the device.
    ADD COLUMN install_id          UUID        NOT NULL,
    -- IANA zone the server needs to turn notify_time into an instant.
    ADD COLUMN time_zone           TEXT        NOT NULL,
    -- Set when the provider rejected the token; cleared by the next registration.
    ADD COLUMN token_invalid_at    TIMESTAMPTZ NULL,
    -- The scheduled callback that will deliver the next reminder.
    ADD COLUMN next_task_name      TEXT        NULL,
    ADD COLUMN next_fire_at        TIMESTAMPTZ NULL,
    -- Per-device, per-local-date claim: at most one reminder attempt chain per day.
    ADD COLUMN reminder_date       DATE        NULL,
    ADD COLUMN reminder_status     TEXT        NULL CHECK (reminder_status IN ('pending', 'sent', 'skipped', 'failed')),
    ADD COLUMN reminder_attempts   INT         NOT NULL DEFAULT 0,
    ADD COLUMN reminder_claimed_at TIMESTAMPTZ NULL,
    ADD COLUMN reminder_error      TEXT        NULL;

CREATE UNIQUE INDEX IF NOT EXISTS devices_user_install_key ON devices (user_id, install_id);
-- A push token belongs to one physical app install; re-registering it under another
-- user moves it (the previous owner's row is deleted), so it is globally unique.
CREATE UNIQUE INDEX IF NOT EXISTS devices_push_token_key ON devices (push_token);
