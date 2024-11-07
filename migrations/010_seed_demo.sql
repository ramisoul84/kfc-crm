-- Demo data so a fresh environment is usable end to end:
-- sign in to CRM, pair the devices, order from the kiosk or website.
--
-- Every insert is ON CONFLICT DO NOTHING, so this never overwrites data
-- you have edited in CRM.
--
-- Accounts (change these passwords on any shared environment):
--   admin@kfc.local          / Admin@12345   super_admin
--   manager.msk001@kfc.local / Admin@12345   restaurant_manager, MSK-001

INSERT INTO users (id, email, password_hash, first_name, last_name, role, restaurant_id, is_active, profile_completed) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@kfc.local',
     '$2a$12$eVT1YCVvfZmdjGrdTtN24OFGg6PL4TNr45xXDuUhGXryndc/gdmDW',
     'Platform', 'Admin', 'super_admin', NULL, true, true),
    ('00000000-0000-0000-0000-000000000002', 'manager.msk001@kfc.local',
     '$2a$12$eVT1YCVvfZmdjGrdTtN24OFGg6PL4TNr45xXDuUhGXryndc/gdmDW',
     'Tverskaya', 'Manager', 'restaurant_manager', '00000000-0000-0000-0000-000000000201', true, true)
ON CONFLICT (id) DO NOTHING;

-- ─── Menu ───────────────────────────────────────────────────────────

INSERT INTO menu_categories (id, name, code, description, sort_order) VALUES
    ('00000000-0000-0000-0000-000000001006', 'Combos', 'COMBOS', 'Burger, side and drink', 5)
ON CONFLICT (id) DO NOTHING;

INSERT INTO menu_items (id, category_id, product_code, name, description, base_price, sort_order) VALUES
    -- Combos
    ('00000000-0000-0000-0000-000000002101', '00000000-0000-0000-0000-000000001006', 'KFC-COMBO-ZINGER', 'Zinger Combo',        'Zinger Burger, fries and a drink',        449.00, 10),
    ('00000000-0000-0000-0000-000000002102', '00000000-0000-0000-0000-000000001006', 'KFC-COMBO-TOWER',  'Tower Combo',         'Tower Burger, fries and a drink',         519.00, 20),
    ('00000000-0000-0000-0000-000000002103', '00000000-0000-0000-0000-000000001006', 'KFC-COMBO-STRIPS', 'Strips Combo',        '5 crispy strips, fries and a drink',      429.00, 30),
    -- Burgers
    ('00000000-0000-0000-0000-000000002011', '00000000-0000-0000-0000-000000001001', 'KFC-CHBURGER-001', 'Chicken Burger',      'Crispy fillet, lettuce and mayo',          159.00, 5),
    ('00000000-0000-0000-0000-000000002012', '00000000-0000-0000-0000-000000001001', 'KFC-TOWER-001',    'Tower Burger',        'Fillet, hash brown, cheese and salsa',     369.00, 30),
    ('00000000-0000-0000-0000-000000002013', '00000000-0000-0000-0000-000000001001', 'KFC-CHEF-001',     'Chefburger',          'Fillet with tomato and signature sauce',   239.00, 40),
    -- Chicken
    ('00000000-0000-0000-0000-000000002021', '00000000-0000-0000-0000-000000001002', 'KFC-STRIPS-003',   'Strips, 3 pcs',       'Hand-breaded chicken strips',              219.00, 20),
    ('00000000-0000-0000-0000-000000002022', '00000000-0000-0000-0000-000000001002', 'KFC-WINGS-006',    'Hot Wings, 6 pcs',    'Spicy chicken wings',                      259.00, 30),
    ('00000000-0000-0000-0000-000000002023', '00000000-0000-0000-0000-000000001002', 'KFC-NUGGETS-009',  'Nuggets, 9 pcs',      'With your choice of sauce',                229.00, 40),
    ('00000000-0000-0000-0000-000000002024', '00000000-0000-0000-0000-000000001002', 'KFC-POPCORN-001',  'Popcorn Chicken',     'Bite-size crispy chicken',                 179.00, 50),
    -- Sides
    ('00000000-0000-0000-0000-000000002031', '00000000-0000-0000-0000-000000001003', 'KFC-COLESLAW-001', 'Coleslaw',            'Fresh cabbage and carrot salad',            99.00, 20),
    ('00000000-0000-0000-0000-000000002032', '00000000-0000-0000-0000-000000001003', 'KFC-CORN-001',     'Corn on the Cob',     'Buttered sweet corn',                      109.00, 30),
    ('00000000-0000-0000-0000-000000002033', '00000000-0000-0000-0000-000000001003', 'KFC-POTATO-001',   'Country Potatoes',    'Seasoned potato wedges',                   129.00, 40),
    -- Drinks
    ('00000000-0000-0000-0000-000000002041', '00000000-0000-0000-0000-000000001004', 'KFC-SPRITE-001',   'Sprite',              'Chilled Sprite',                           109.00, 20),
    ('00000000-0000-0000-0000-000000002042', '00000000-0000-0000-0000-000000001004', 'KFC-ICETEA-001',   'Ice Tea Lemon',       'Iced black tea with lemon',                129.00, 30),
    ('00000000-0000-0000-0000-000000002043', '00000000-0000-0000-0000-000000001004', 'KFC-AMERICANO',    'Americano',           'Freshly brewed coffee',                    119.00, 40),
    ('00000000-0000-0000-0000-000000002044', '00000000-0000-0000-0000-000000001004', 'KFC-WATER-001',    'Still Water',         '0.5 l',                                     79.00, 50),
    -- Desserts
    ('00000000-0000-0000-0000-000000002051', '00000000-0000-0000-0000-000000001005', 'KFC-PIE-001',      'Cherry Pie',          'Hot pie with cherry filling',               89.00, 20),
    ('00000000-0000-0000-0000-000000002052', '00000000-0000-0000-0000-000000001005', 'KFC-SHAKE-001',    'Milkshake',           'Vanilla, strawberry or chocolate',         179.00, 30)
ON CONFLICT (id) DO NOTHING;

INSERT INTO menu_item_variations (id, menu_item_id, name, price_delta, is_default, sort_order) VALUES
    -- French Fries
    ('00000000-0000-0000-0000-000000003011', '00000000-0000-0000-0000-000000002004', 'Standard', 0,  true,  10),
    ('00000000-0000-0000-0000-000000003012', '00000000-0000-0000-0000-000000002004', 'Large',    40, false, 20),
    -- Sprite
    ('00000000-0000-0000-0000-000000003021', '00000000-0000-0000-0000-000000002041', '0.3 l',    0,  false, 10),
    ('00000000-0000-0000-0000-000000003022', '00000000-0000-0000-0000-000000002041', '0.4 l',    30, true,  20),
    ('00000000-0000-0000-0000-000000003023', '00000000-0000-0000-0000-000000002041', '0.5 l',    60, false, 30),
    -- Combos: drink choice
    ('00000000-0000-0000-0000-000000003031', '00000000-0000-0000-0000-000000002101', 'with Coca-Cola', 0, true,  10),
    ('00000000-0000-0000-0000-000000003032', '00000000-0000-0000-0000-000000002101', 'with Sprite',    0, false, 20),
    ('00000000-0000-0000-0000-000000003033', '00000000-0000-0000-0000-000000002101', 'with Ice Tea',  20, false, 30),
    ('00000000-0000-0000-0000-000000003041', '00000000-0000-0000-0000-000000002102', 'with Coca-Cola', 0, true,  10),
    ('00000000-0000-0000-0000-000000003042', '00000000-0000-0000-0000-000000002102', 'with Sprite',    0, false, 20),
    ('00000000-0000-0000-0000-000000003051', '00000000-0000-0000-0000-000000002103', 'with Coca-Cola', 0, true,  10),
    ('00000000-0000-0000-0000-000000003052', '00000000-0000-0000-0000-000000002103', 'with Sprite',    0, false, 20),
    -- Milkshake flavours
    ('00000000-0000-0000-0000-000000003061', '00000000-0000-0000-0000-000000002052', 'Vanilla',      0, true,  10),
    ('00000000-0000-0000-0000-000000003062', '00000000-0000-0000-0000-000000002052', 'Strawberry',   0, false, 20),
    ('00000000-0000-0000-0000-000000003063', '00000000-0000-0000-0000-000000002052', 'Chocolate',    0, false, 30)
ON CONFLICT (id) DO NOTHING;

-- ─── Devices ────────────────────────────────────────────────────────
-- Inactive until paired from CRM (Devices → Pair).

INSERT INTO devices (id, restaurant_id, type, serial_number, is_active) VALUES
    -- MSK-001 Tverskaya
    ('00000000-0000-0000-0000-000000004001', '00000000-0000-0000-0000-000000000201', 'kiosk',         'KIOSK-MSK001-01',   false),
    ('00000000-0000-0000-0000-000000004002', '00000000-0000-0000-0000-000000000201', 'kiosk',         'KIOSK-MSK001-02',   false),
    ('00000000-0000-0000-0000-000000004003', '00000000-0000-0000-0000-000000000201', 'kds',           'KDS-MSK001-01',     false),
    ('00000000-0000-0000-0000-000000004004', '00000000-0000-0000-0000-000000000201', 'order_display', 'DISPLAY-MSK001-01', false),
    -- MSK-002 Arbat
    ('00000000-0000-0000-0000-000000004011', '00000000-0000-0000-0000-000000000202', 'kiosk',         'KIOSK-MSK002-01',   false),
    ('00000000-0000-0000-0000-000000004012', '00000000-0000-0000-0000-000000000202', 'kds',           'KDS-MSK002-01',     false),
    ('00000000-0000-0000-0000-000000004013', '00000000-0000-0000-0000-000000000202', 'order_display', 'DISPLAY-MSK002-01', false)
ON CONFLICT DO NOTHING;
