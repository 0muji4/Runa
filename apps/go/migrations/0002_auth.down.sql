DROP TABLE IF EXISTS refresh_tokens;

DROP INDEX IF EXISTS users_google_sub_key;
DROP INDEX IF EXISTS users_apple_sub_key;
DROP INDEX IF EXISTS users_email_key;

ALTER TABLE users
    DROP COLUMN IF EXISTS premium_expires_at,
    DROP COLUMN IF EXISTS is_premium,
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS display_name,
    DROP COLUMN IF EXISTS google_sub,
    DROP COLUMN IF EXISTS apple_sub,
    DROP COLUMN IF EXISTS auth_provider,
    DROP COLUMN IF EXISTS email;
