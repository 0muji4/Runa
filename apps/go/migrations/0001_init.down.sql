-- 0001_init.down.sql
-- Reverses 0001_init.up.sql. The pgcrypto extension is left in place because it
-- may be shared by other objects; only the table this migration created is dropped.

DROP TABLE IF EXISTS users;
