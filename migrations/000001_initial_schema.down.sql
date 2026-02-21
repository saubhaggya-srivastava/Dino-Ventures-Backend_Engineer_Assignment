-- This file will contain rollback commands

-- Drop view
DROP VIEW IF EXISTS account_balances;

-- Drop triggers
DROP TRIGGER IF EXISTS prevent_ledger_delete ON ledger_entries;
DROP TRIGGER IF EXISTS prevent_ledger_update ON ledger_entries;
DROP FUNCTION IF EXISTS prevent_ledger_modification();

-- Drop indexes
DROP INDEX IF EXISTS idx_ledger_transaction;
DROP INDEX IF EXISTS idx_ledger_account_asset;
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_idempotency;

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS asset_types;
