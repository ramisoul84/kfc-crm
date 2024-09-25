-- Restaurants table
CREATE TABLE IF NOT EXISTS restaurants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(200) NOT NULL,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    region_id   UUID         NOT NULL REFERENCES regions(id) ON DELETE RESTRICT,
    address     TEXT,
    city        VARCHAR(100),
    postal_code VARCHAR(10),
    phone       VARCHAR(20),
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_restaurants_region
    ON restaurants (region_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_restaurants_code
    ON restaurants (code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_restaurants_active
    ON restaurants (is_active)
    WHERE deleted_at IS NULL;

-- Seed data: a few restaurants per major region
INSERT INTO restaurants (id, name, code, region_id, address, city, phone) VALUES
    -- Moscow
    ('00000000-0000-0000-0000-000000000201', 'KFC Moscow Tverskaya', 'MSK-001', '00000000-0000-0000-0000-000000000101', 'Tverskaya St, 1',     'Moscow', '+7 (495) 123-4501'),
    ('00000000-0000-0000-0000-000000000202', 'KFC Moscow Arbat',     'MSK-002', '00000000-0000-0000-0000-000000000101', 'Arbat St, 15',         'Moscow', '+7 (495) 123-4502'),
    ('00000000-0000-0000-0000-000000000203', 'KFC Moscow Red Square','MSK-003', '00000000-0000-0000-0000-000000000101', 'Nikolskaya St, 5',     'Moscow', '+7 (495) 123-4503'),

    -- Saint Petersburg
    ('00000000-0000-0000-0000-000000000204', 'KFC SPB Nevsky',       'SPB-001', '00000000-0000-0000-0000-000000000102', 'Nevsky Prospekt, 50',  'Saint Petersburg', '+7 (812) 123-4501'),
    ('00000000-0000-0000-0000-000000000205', 'KFC SPB Vosstaniya',   'SPB-002', '00000000-0000-0000-0000-000000000102', 'Vosstaniya St, 10',    'Saint Petersburg', '+7 (812) 123-4502'),

    -- Novosibirsk
    ('00000000-0000-0000-0000-000000000206', 'KFC Novosibirsk Center','NVS-001', '00000000-0000-0000-0000-000000000103', 'Krasny Prospekt, 25',  'Novosibirsk', '+7 (383) 123-4501'),

    -- Yekaterinburg
    ('00000000-0000-0000-0000-000000000207', 'KFC Yekaterinburg Center','EKB-001', '00000000-0000-0000-0000-000000000104', 'Lenina St, 30',        'Yekaterinburg', '+7 (343) 123-4501'),

    -- Kazan
    ('00000000-0000-0000-0000-000000000208', 'KFC Kazan Bauman',     'KZN-001', '00000000-0000-0000-0000-000000000105', 'Baumana St, 20',       'Kazan', '+7 (843) 123-4501')
ON CONFLICT (id) DO NOTHING;