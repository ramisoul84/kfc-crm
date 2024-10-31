CREATE TABLE IF NOT EXISTS menu_item_variations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_item_id UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    price_delta  DECIMAL(10,2) NOT NULL DEFAULT 0,
    is_default   BOOLEAN DEFAULT false,
    sort_order   INT DEFAULT 0,
    is_active    BOOLEAN DEFAULT true,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_menu_item_variations_item
    ON menu_item_variations (menu_item_id) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_menu_item_variations_default
    ON menu_item_variations (menu_item_id) WHERE is_default = true AND is_active = true;

-- Seed: sizes for Coca-Cola
INSERT INTO menu_item_variations (id, menu_item_id, name, price_delta, is_default, sort_order) VALUES
    ('00000000-0000-0000-0000-000000003001', '00000000-0000-0000-0000-000000002005', 'Small',   0,    false, 10),
    ('00000000-0000-0000-0000-000000003002', '00000000-0000-0000-0000-000000002005', 'Medium',  50,   true,  20),
    ('00000000-0000-0000-0000-000000003003', '00000000-0000-0000-0000-000000002005', 'Large',   100,  false, 30)
ON CONFLICT (id) DO NOTHING;