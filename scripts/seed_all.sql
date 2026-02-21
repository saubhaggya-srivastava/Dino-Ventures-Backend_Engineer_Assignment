-- Master seed script placeholder
-- Add the master seed script content here

-- Master seed script
-- Runs all seed scripts in the correct order
-- Execute this after running database migrations

-- 1. Create asset types first (no dependencies)
\i seed_asset_types.sql

-- 2. Create accounts (depends on nothing)
\i seed_accounts.sql

-- 3. Create initial balances (depends on asset_types and accounts)
\i seed_initial_balances.sql

-- Verify the seeding worked
SELECT 'Asset Types:' as info;
SELECT id, name FROM asset_types;

SELECT 'Accounts:' as info;
SELECT id, type, owner_id FROM accounts ORDER BY id;

SELECT 'Account Balances:' as info;
SELECT 
    a.owner_id,
    ab.asset_type_id,
    ab.balance
FROM account_balances ab
JOIN accounts a ON ab.account_id = a.id
ORDER BY a.id, ab.asset_type_id;
