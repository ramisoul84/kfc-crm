CREATE TABLE IF NOT EXISTS devices (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id UUID NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    type          VARCHAR(20) NOT NULL,
    serial_number VARCHAR(100) NOT NULL UNIQUE,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT chk_devices_type CHECK (
        type IN ('pos', 'kiosk', 'kds', 'order_display')
    )
);

CREATE INDEX IF NOT EXISTS idx_devices_restaurant
    ON devices (restaurant_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_devices_type
    ON devices (type)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_devices_serial
    ON devices (serial_number)
    WHERE deleted_at IS NULL;