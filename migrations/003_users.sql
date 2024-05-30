-- Users table
CREATE TABLE IF NOT EXISTS users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              VARCHAR(255) NOT NULL UNIQUE,
    password_hash      VARCHAR(255) NOT NULL,
    first_name         VARCHAR(100),
    last_name          VARCHAR(100),
    phone              VARCHAR(20),
    role               VARCHAR(50)  NOT NULL,
    region_id          UUID REFERENCES regions(id)     ON DELETE SET NULL,
    restaurant_id      UUID REFERENCES restaurants(id) ON DELETE SET NULL,
    is_active          BOOLEAN      NOT NULL DEFAULT true,
    profile_completed  BOOLEAN      NOT NULL DEFAULT false,
    last_login_at      TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ,

    -- Role validation at the DB level
    CONSTRAINT chk_users_role CHECK (
        role IN ('super_admin', 'regional_manager', 'restaurant_manager', 'shift_manager', 'cashier')
    )
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users (email)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_role
    ON users (role)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_region
    ON users (region_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_restaurant
    ON users (restaurant_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_active
    ON users (is_active)
    WHERE deleted_at IS NULL;