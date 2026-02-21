-- This file will contain asset type seed data

-- Uses INSERT ... ON CONFLICT DO NOTHING for idempotent seeding

INSERT INTO asset_types (id, name, description) VALUES
    ('gold_coins', 'Gold Coins', 'Premium currency for purchasing items and upgrades'),
    ('diamonds', 'Diamonds', 'Rare currency for exclusive content and features'),
    ('loyalty_points', 'Loyalty Points', 'Earned through gameplay and daily activities')
ON CONFLICT (id) DO NOTHING;

