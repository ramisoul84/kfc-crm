-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Regions table
CREATE TABLE IF NOT EXISTS regions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(10)  NOT NULL UNIQUE,
    timezone    VARCHAR(50)  NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_regions_active
    ON regions (is_active)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_regions_code
    ON regions (code)
    WHERE deleted_at IS NULL;

-- Seed data: major Russian cities
INSERT INTO regions (id, name, code, timezone) VALUES
    ('00000000-0000-0000-0000-000000000101', 'Moscow',           'MSK', 'Europe/Moscow'),
    ('00000000-0000-0000-0000-000000000102', 'Saint Petersburg', 'SPB', 'Europe/Moscow'),
    ('00000000-0000-0000-0000-000000000103', 'Novosibirsk',      'NVS', 'Asia/Novosibirsk'),
    ('00000000-0000-0000-0000-000000000104', 'Yekaterinburg',    'EKB', 'Asia/Yekaterinburg'),
    ('00000000-0000-0000-0000-000000000105', 'Kazan',            'KZN', 'Europe/Moscow'),
    ('00000000-0000-0000-0000-000000000106', 'Nizhny Novgorod',  'NNG', 'Europe/Moscow'),
    ('00000000-0000-0000-0000-000000000107', 'Chelyabinsk',      'CLB', 'Asia/Yekaterinburg'),
    ('00000000-0000-0000-0000-000000000108', 'Samara',           'SMR', 'Europe/Samara'),
    ('00000000-0000-0000-0000-000000000109', 'Omsk',             'OMK', 'Asia/Omsk'),
    ('00000000-0000-0000-0000-000000000110', 'Rostov-on-Don',    'RND', 'Europe/Moscow')
ON CONFLICT (id) DO NOTHING;