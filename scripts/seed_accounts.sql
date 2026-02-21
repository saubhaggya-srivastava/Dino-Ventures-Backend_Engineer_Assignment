-- This file will contain account seed data

-- Seed data for accounts
-- Creates system accounts and sample user accounts
-- Uses INSERT ... ON CONFLICT DO NOTHING for idempotent seeding

-- System accounts
INSERT INTO accounts (type, owner_id) VALUES
    ('system', 'treasury'),
    ('system', 'revenue')
ON CONFLICT (type, owner_id) DO NOTHING;

-- Sample user accounts
INSERT INTO accounts (type, owner_id) VALUES
    ('user', 'user_001'),
    ('user', 'user_002'),
    ('user', 'user_003')
ON CONFLICT (type, owner_id) DO NOTHING;
