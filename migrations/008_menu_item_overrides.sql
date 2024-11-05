CREATE TABLE IF NOT EXISTS restaurant_menu_overrides (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id         UUID NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    menu_item_id          UUID NOT NULL REFERENCES menu_items(id)  ON DELETE CASCADE,
    price_override        DECIMAL(10,2),
    is_available_override BOOLEAN,
    reason                TEXT,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW(),

    -- One override per restaurant-item pair
    UNIQUE (restaurant_id, menu_item_id),

    -- At least one override field must be set
    CONSTRAINT chk_override_has_value CHECK (
        price_override IS NOT NULL OR is_available_override IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_menu_overrides_restaurant
    ON restaurant_menu_overrides (restaurant_id);

CREATE INDEX IF NOT EXISTS idx_menu_overrides_item
    ON restaurant_menu_overrides (menu_item_id);