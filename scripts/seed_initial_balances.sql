-- This file will contain initial balance seed data

-- Seed data for initial balances
-- Creates initial balances by inserting ledger entries
-- This demonstrates the correct way to set balances in our ledger system

-- First, we need to create dummy transactions for the initial balance entries
-- These represent "system initialization" transactions

INSERT INTO transactions (idempotency_key, type, asset_type_id, amount, from_account_id, to_account_id, metadata, status, created_at, completed_at) 
SELECT 
    'init_treasury_' || at.id,
    'bonus',
    at.id,
    1000000,  -- 1 million units
    1,  -- treasury account (will be ID 1)
    1,  -- treasury account (self-transaction for initialization)
    '{"description": "Initial treasury balance"}',
    'completed',
    NOW(),
    NOW()
FROM asset_types at
WHERE NOT EXISTS (
    SELECT 1 FROM transactions WHERE idempotency_key = 'init_treasury_' || at.id
);

-- Create ledger entries for treasury initial balances
INSERT INTO ledger_entries (transaction_id, account_id, asset_type_id, amount, created_at)
SELECT 
    t.id,
    1,  -- treasury account ID
    t.asset_type_id,
    1000000,  -- 1 million units
    NOW()
FROM transactions t
WHERE t.idempotency_key LIKE 'init_treasury_%'
AND NOT EXISTS (
    SELECT 1 FROM ledger_entries le 
    WHERE le.transaction_id = t.id AND le.account_id = 1
);

-- Create initial balances for sample users
-- User 1: 5000 gold_coins, 100 diamonds, 1000 loyalty_points
INSERT INTO transactions (idempotency_key, type, asset_type_id, amount, from_account_id, to_account_id, metadata, status, created_at, completed_at)
VALUES 
    ('init_user_001_gold_coins', 'bonus', 'gold_coins', 5000, 1, 3, '{"description": "Welcome bonus"}', 'completed', NOW(), NOW()),
    ('init_user_001_diamonds', 'bonus', 'diamonds', 100, 1, 3, '{"description": "Welcome bonus"}', 'completed', NOW(), NOW()),
    ('init_user_001_loyalty_points', 'bonus', 'loyalty_points', 1000, 1, 3, '{"description": "Welcome bonus"}', 'completed', NOW(), NOW())
ON CONFLICT (idempotency_key) DO NOTHING;

-- Create corresponding ledger entries for user 1
INSERT INTO ledger_entries (transaction_id, account_id, asset_type_id, amount, created_at)
SELECT t.id, 3, t.asset_type_id, t.amount, NOW()
FROM transactions t
WHERE t.idempotency_key LIKE 'init_user_001_%'
AND NOT EXISTS (SELECT 1 FROM ledger_entries le WHERE le.transaction_id = t.id AND le.account_id = 3);

-- Debit entries from treasury (double-entry bookkeeping)
INSERT INTO ledger_entries (transaction_id, account_id, asset_type_id, amount, created_at)
SELECT t.id, 1, t.asset_type_id, -t.amount, NOW()
FROM transactions t
WHERE t.idempotency_key LIKE 'init_user_001_%'
AND NOT EXISTS (SELECT 1 FROM ledger_entries le WHERE le.transaction_id = t.id AND le.account_id = 1);
