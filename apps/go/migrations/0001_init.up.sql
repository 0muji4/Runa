-- 0001_init.up.sql
-- Minimal users table stub for the walking skeleton. No feature columns yet;
-- the schema grows alongside the product. gen_random_uuid() requires pgcrypto.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
