-- Sanchay IMS — PostgreSQL schema
-- Run against the sanchay-ims database:
--   psql -U postgres -d sanchay-ims -f schema.sql

-- gen_random_uuid() is built-in since PostgreSQL 13+

-- ── Schemas ──────────────────────────────────────────────────────────────────
-- Auth/account objects live in "users".
-- Warehouse/location setup objects live in "locations".
-- Product inventory objects live in "stocks".
-- Receipts and delivery operations live in "operations".

CREATE SCHEMA IF NOT EXISTS "users";
CREATE SCHEMA IF NOT EXISTS "locations";
CREATE SCHEMA IF NOT EXISTS "stocks";
CREATE SCHEMA IF NOT EXISTS "operations";

-- Let the postgres role use and create within this schema.
GRANT USAGE  ON SCHEMA "users" TO postgres;
GRANT CREATE ON SCHEMA "users" TO postgres;
GRANT USAGE  ON SCHEMA "locations" TO postgres;
GRANT CREATE ON SCHEMA "locations" TO postgres;
GRANT USAGE  ON SCHEMA "stocks" TO postgres;
GRANT CREATE ON SCHEMA "stocks" TO postgres;
GRANT USAGE  ON SCHEMA "operations" TO postgres;
GRANT CREATE ON SCHEMA "operations" TO postgres;

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

-- Move older settings tables from "users" schema to "locations" schema when
-- upgrading an existing database.
DO $$
BEGIN
    IF to_regclass('users.warehouses') IS NOT NULL AND to_regclass('locations.warehouses') IS NULL THEN
        EXECUTE 'ALTER TABLE "users".warehouses SET SCHEMA "locations"';
    END IF;

    IF to_regclass('users.locations') IS NOT NULL AND to_regclass('locations.locations') IS NULL THEN
        EXECUTE 'ALTER TABLE "users".locations SET SCHEMA "locations"';
    END IF;

    IF to_regclass('users.location_warehouses') IS NOT NULL AND to_regclass('locations.location_warehouses') IS NULL THEN
        EXECUTE 'ALTER TABLE "users".location_warehouses SET SCHEMA "locations"';
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS "locations".warehouses (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(120) NOT NULL,
    short_code   VARCHAR(30)  NOT NULL UNIQUE,
    address      TEXT,
    description  TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouses_name       ON "locations".warehouses (name);
CREATE INDEX IF NOT EXISTS idx_warehouses_short_code ON "locations".warehouses (short_code);

CREATE TABLE IF NOT EXISTS "locations".locations (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(120) NOT NULL,
    short_code   VARCHAR(30)  NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_locations_name       ON "locations".locations (name);
CREATE INDEX IF NOT EXISTS idx_locations_short_code ON "locations".locations (short_code);

-- Join table that maps one location to one or many warehouses.
CREATE TABLE IF NOT EXISTS "locations".location_warehouses (
    location_id   UUID        NOT NULL REFERENCES "locations".locations(id) ON DELETE CASCADE,
    warehouse_id  UUID        NOT NULL REFERENCES "locations".warehouses(id) ON DELETE RESTRICT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (location_id, warehouse_id)
);

CREATE INDEX IF NOT EXISTS idx_location_warehouses_location_id  ON "locations".location_warehouses (location_id);
CREATE INDEX IF NOT EXISTS idx_location_warehouses_warehouse_id ON "locations".location_warehouses (warehouse_id);

CREATE TABLE IF NOT EXISTS "stocks".categories (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(120) NOT NULL UNIQUE,
    description  TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stocks_categories_name ON "stocks".categories (name);

INSERT INTO "stocks".categories (name, description)
VALUES
    ('Raw Material', 'Materials used as input for production'),
    ('Finished Goods', 'Final sellable inventory items'),
    ('Consumables', 'Operational use items with regular consumption'),
    ('Packaging', 'Packaging and handling inventory')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS "stocks".products (
    id                     UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    sku                    VARCHAR(80)    NOT NULL UNIQUE DEFAULT ('SKU/' || UPPER(SUBSTRING(REPLACE(gen_random_uuid()::text, '-', '') FROM 1 FOR 12))),
    name                   VARCHAR(180)   NOT NULL,
    cost                   NUMERIC(12,2)  NOT NULL CHECK (cost >= 0),
    on_hand_quantity       INTEGER        NOT NULL CHECK (on_hand_quantity >= 0),
    free_to_use_quantity   INTEGER        NOT NULL CHECK (free_to_use_quantity >= 0),
    category_id            UUID           NOT NULL REFERENCES "stocks".categories(id) ON DELETE RESTRICT,
    location_id            UUID           NOT NULL REFERENCES "locations".locations(id) ON DELETE RESTRICT,
    description            TEXT,
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CHECK (free_to_use_quantity <= on_hand_quantity)
);

CREATE INDEX IF NOT EXISTS idx_stocks_products_name         ON "stocks".products (name);
CREATE INDEX IF NOT EXISTS idx_stocks_products_sku          ON "stocks".products (sku);
CREATE INDEX IF NOT EXISTS idx_stocks_products_category_id  ON "stocks".products (category_id);
CREATE INDEX IF NOT EXISTS idx_stocks_products_location_id  ON "stocks".products (location_id);
CREATE INDEX IF NOT EXISTS idx_stocks_products_updated_at   ON "stocks".products (updated_at DESC);

-- Supports many-to-many mapping between products and locations with per-location
-- quantity tracking.
CREATE TABLE IF NOT EXISTS "stocks".product_stock_levels (
    product_id             UUID           NOT NULL REFERENCES "stocks".products(id) ON DELETE CASCADE,
    location_id            UUID           NOT NULL REFERENCES "locations".locations(id) ON DELETE RESTRICT,
    on_hand_quantity       INTEGER        NOT NULL CHECK (on_hand_quantity >= 0),
    free_to_use_quantity   INTEGER        NOT NULL CHECK (free_to_use_quantity >= 0),
    created_at             TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_id, location_id),
    CHECK (free_to_use_quantity <= on_hand_quantity)
);

CREATE INDEX IF NOT EXISTS idx_product_stock_levels_product_id  ON "stocks".product_stock_levels (product_id);
CREATE INDEX IF NOT EXISTS idx_product_stock_levels_location_id ON "stocks".product_stock_levels (location_id);
CREATE INDEX IF NOT EXISTS idx_product_stock_levels_updated_at  ON "stocks".product_stock_levels (updated_at DESC);

-- Backfill the new stock-level table from legacy single-location columns.
INSERT INTO "stocks".product_stock_levels (
    product_id,
    location_id,
    on_hand_quantity,
    free_to_use_quantity
)
SELECT
    p.id,
    p.location_id,
    p.on_hand_quantity,
    p.free_to_use_quantity
FROM "stocks".products p
ON CONFLICT (product_id, location_id) DO UPDATE
SET
    on_hand_quantity = EXCLUDED.on_hand_quantity,
    free_to_use_quantity = EXCLUDED.free_to_use_quantity,
    updated_at = NOW();

CREATE SEQUENCE IF NOT EXISTS "operations".reference_seq START WITH 1 INCREMENT BY 1;

CREATE TABLE IF NOT EXISTS "operations".orders (
    id                    BIGSERIAL      PRIMARY KEY,
    reference_sequence    BIGINT         NOT NULL UNIQUE,
    reference_number      VARCHAR(120)   NOT NULL UNIQUE,
    operation_type        VARCHAR(3)     NOT NULL CHECK (operation_type IN ('IN', 'OUT')),
    from_party            VARCHAR(180),
    to_party              VARCHAR(180),
    location_id           UUID           NOT NULL REFERENCES "locations".locations(id) ON DELETE RESTRICT,
    warehouse_short_code  VARCHAR(30)    NOT NULL,
    contact_number        VARCHAR(32),
    scheduled_date        DATE           NOT NULL,
    status                VARCHAR(20)    NOT NULL CHECK (status IN ('DRAFT', 'WAITING', 'READY', 'DONE', 'CANCELLED')),
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Ensure status check includes CANCELLED for already-created tables.
DO $$
DECLARE
    status_constraint RECORD;
BEGIN
    FOR status_constraint IN
        SELECT c.conname
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = 'operations'
          AND t.relname = 'orders'
          AND c.contype = 'c'
          AND pg_get_constraintdef(c.oid) ILIKE '%status%'
    LOOP
        EXECUTE format('ALTER TABLE "operations".orders DROP CONSTRAINT %I', status_constraint.conname);
    END LOOP;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = 'operations'
          AND t.relname = 'orders'
          AND c.conname = 'operations_orders_status_check'
    ) THEN
        EXECUTE '
            ALTER TABLE "operations".orders
            ADD CONSTRAINT operations_orders_status_check
            CHECK (status IN (''DRAFT'', ''WAITING'', ''READY'', ''DONE'', ''CANCELLED''))
        ';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_operations_orders_reference_number ON "operations".orders (reference_number);
CREATE INDEX IF NOT EXISTS idx_operations_orders_operation_type   ON "operations".orders (operation_type);
CREATE INDEX IF NOT EXISTS idx_operations_orders_location_id      ON "operations".orders (location_id);
CREATE INDEX IF NOT EXISTS idx_operations_orders_status           ON "operations".orders (status);
CREATE INDEX IF NOT EXISTS idx_operations_orders_scheduled_date   ON "operations".orders (scheduled_date DESC);
CREATE INDEX IF NOT EXISTS idx_operations_orders_created_at       ON "operations".orders (created_at DESC);

CREATE TABLE IF NOT EXISTS "operations".order_items (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      BIGINT      NOT NULL REFERENCES "operations".orders(id) ON DELETE CASCADE,
    product_id    UUID        NOT NULL REFERENCES "stocks".products(id) ON DELETE RESTRICT,
    quantity      INTEGER     NOT NULL CHECK (quantity > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_operations_order_items_order_id   ON "operations".order_items (order_id);
CREATE INDEX IF NOT EXISTS idx_operations_order_items_product_id ON "operations".order_items (product_id);
