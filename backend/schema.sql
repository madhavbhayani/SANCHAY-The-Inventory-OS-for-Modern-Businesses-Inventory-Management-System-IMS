-- Sanchay IMS — PostgreSQL schema
-- Run against the sanchay-ims database:
--   psql -U postgres -d sanchay-ims -f schema.sql

-- gen_random_uuid() is built-in since PostgreSQL 13+

-- ── Schema ───────────────────────────────────────────────────────────────────
-- All Sanchay IMS objects live in the "users" schema, not the default public.

CREATE SCHEMA IF NOT EXISTS "users";

-- Let the postgres role use and create within this schema.
GRANT USAGE  ON SCHEMA "users" TO postgres;
GRANT CREATE ON SCHEMA "users" TO postgres;

-- ── Drop old public-schema tables if they were created before this migration ─
DROP TABLE IF EXISTS public.login_history;
DROP TABLE IF EXISTS public.users;

-- ── Tables ───────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS "users".users (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    login_id    VARCHAR(64)  UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    password    VARCHAR(255) NOT NULL,           -- bcrypt hash, never plain text
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_login_id ON "users".users (login_id);
CREATE INDEX IF NOT EXISTS idx_users_email    ON "users".users (lower(email));

CREATE TABLE IF NOT EXISTS "users".login_history (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         REFERENCES "users".users(id) ON DELETE CASCADE,  -- NULL on unknown user
    ip_address     VARCHAR(64),
    user_agent     TEXT,
    browser        VARCHAR(128),
    os             VARCHAR(128),
    success        BOOLEAN      NOT NULL DEFAULT TRUE,
    failure_reason VARCHAR(255),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_login_history_user_id    ON "users".login_history (user_id);
CREATE INDEX IF NOT EXISTS idx_login_history_created_at ON "users".login_history (created_at DESC);
