CREATE TABLE IF NOT EXISTS menu_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id  UUID NOT NULL REFERENCES menu_categories(id) ON DELETE RESTRICT,
    product_code VARCHAR(50)  UNIQUE NOT NULL,
    name         VARCHAR(200) NOT NULL,
    description  TEXT,
    image_url    TEXT,
    base_price   DECIMAL(10,2) NOT NULL,
    sort_order   INT DEFAULT 0,
    is_active    BOOLEAN DEFAULT true,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_menu_items_category
    ON menu_items (category_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_menu_items_code
    ON menu_items (product_code) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_menu_items_active
    ON menu_items (is_active) WHERE deleted_at IS NULL;

-- Seed a few items
INSERT INTO menu_items (id, category_id, product_code, name, description, base_price, sort_order) VALUES
    ('00000000-0000-0000-0000-000000002001', '00000000-0000-0000-0000-000000001001', 'KFC-ZINGER-001',  'Zinger Burger',      'Spicy crispy chicken burger',  299.00, 10),
    ('00000000-0000-0000-0000-000000002002', '00000000-0000-0000-0000-000000001001', 'KFC-DOUBLE-001',  'Double Zinger',      'Double patty zinger burger',   449.00, 20),
    ('00000000-0000-0000-0000-000000002003', '00000000-0000-0000-0000-000000001002', 'KFC-BUCKET-008',  'Bucket 8 Pieces',    '8 pieces of fried chicken',    799.00, 10),
    ('00000000-0000-0000-0000-000000002004', '00000000-0000-0000-0000-000000001003', 'KFC-FRIES-001',   'French Fries',       'Crispy golden fries',          119.00, 10),
    ('00000000-0000-0000-0000-000000002005', '00000000-0000-0000-0000-000000001004', 'KFC-COLA-001',    'Coca-Cola',          'Chilled Coca-Cola',            109.00, 10),
    ('00000000-0000-0000-0000-000000002006', '00000000-0000-0000-0000-000000001005', 'KFC-SUNDAE-001',  'Ice Cream Sundae',   'Vanilla sundae with topping',   99.00, 10)
ON CONFLICT (id) DO NOTHING;