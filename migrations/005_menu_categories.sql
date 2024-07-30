CREATE TABLE IF NOT EXISTS menu_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(50)  UNIQUE NOT NULL,
    description TEXT,
    image_url   TEXT,
    sort_order  INT DEFAULT 0,
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_menu_categories_code
    ON menu_categories (code) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_menu_categories_active
    ON menu_categories (is_active) WHERE deleted_at IS NULL;

INSERT INTO menu_categories (id, name, code, description, sort_order) VALUES
    ('00000000-0000-0000-0000-000000001001', 'Burgers',  'BURGERS',  'KFC Burgers',     10),
    ('00000000-0000-0000-0000-000000001002', 'Chicken',  'CHICKEN',  'Fried Chicken',   20),
    ('00000000-0000-0000-0000-000000001003', 'Sides',    'SIDES',    'Fries and sides', 30),
    ('00000000-0000-0000-0000-000000001004', 'Drinks',   'DRINKS',   'Beverages',       40),
    ('00000000-0000-0000-0000-000000001005', 'Desserts', 'DESSERTS', 'Sweet treats',    50)
ON CONFLICT (id) DO NOTHING;