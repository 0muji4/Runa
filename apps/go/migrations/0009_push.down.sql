DROP INDEX IF EXISTS devices_push_token_key;
DROP INDEX IF EXISTS devices_user_install_key;
ALTER TABLE devices
    DROP COLUMN IF EXISTS reminder_error,
    DROP COLUMN IF EXISTS reminder_claimed_at,
    DROP COLUMN IF EXISTS reminder_attempts,
    DROP COLUMN IF EXISTS reminder_status,
    DROP COLUMN IF EXISTS reminder_date,
    DROP COLUMN IF EXISTS next_fire_at,
    DROP COLUMN IF EXISTS next_task_name,
    DROP COLUMN IF EXISTS token_invalid_at,
    DROP COLUMN IF EXISTS time_zone,
    DROP COLUMN IF EXISTS install_id;
CREATE UNIQUE INDEX IF NOT EXISTS devices_user_token_key ON devices (user_id, push_token);
