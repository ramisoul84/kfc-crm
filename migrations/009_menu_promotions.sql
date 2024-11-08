CREATE TABLE IF NOT EXISTS menu_promotions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(200) NOT NULL,
    description    TEXT,
    scope          VARCHAR(20) NOT NULL,
    region_id      UUID REFERENCES regions(id)     ON DELETE CASCADE,
    restaurant_id  UUID REFERENCES restaurants(id) ON DELETE CASCADE,
    discount_type  VARCHAR(20) NOT NULL,
    discount_value DECIMAL(10,2) NOT NULL,
    starts_at      TIMESTAMPTZ NOT NULL,
    ends_at        TIMESTAMPTZ NOT NULL,
    is_active      BOOLEAN DEFAULT true,
    priority       INT DEFAULT 0,
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,

    CONSTRAINT chk_promotion_scope CHECK (
        (scope = 'global'     AND region_id IS NULL     AND restaurant_id IS NULL) OR
        (scope = 'region'     AND region_id IS NOT NULL AND restaurant_id IS NULL) OR
        (scope = 'restaurant' AND region_id IS NULL     AND restaurant_id IS NOT NULL)
    ),
    CONSTRAINT chk_promotion_discount CHECK (
        (discount_type = 'percentage' AND discount_value > 0 AND discount_value <= 100) OR
        (discount_type = 'fixed'      AND discount_value > 0)
    ),
    CONSTRAINT chk_promotion_dates CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_promotions_active
    ON menu_promotions (is_active, starts_at, ends_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_promotions_scope
    ON menu_promotions (scope) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_promotions_region
    ON menu_promotions (region_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_promotions_restaurant
    ON menu_promotions (restaurant_id) WHERE deleted_at IS NULL;

-- Optional item targeting
CREATE TABLE IF NOT EXISTS menu_promotion_items (
    promotion_id UUID NOT NULL REFERENCES menu_promotions(id) ON DELETE CASCADE,
    menu_item_id UUID NOT NULL REFERENCES menu_items(id)      ON DELETE CASCADE,
    PRIMARY KEY (promotion_id, menu_item_id)
);

CREATE INDEX IF NOT EXISTS idx_promotion_items_item
    ON menu_promotion_items (menu_item_id);