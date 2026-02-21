-- Asset types table
CREATE TABLE asset_types(
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Accounts table
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(20) NOT NULL CHECK (type IN ('user', 'system')),
    owner_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(type,owner_id)
);


-- Transactions table
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(100) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('topup', 'bonus', 'spend')),
    asset_type_id VARCHAR(50) NOT NULL REFERENCES asset_types(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    from_account_id BIGINT NOT NULL REFERENCES accounts(id),
    to_account_id BIGINT NOT NULL REFERENCES accounts(id),
    metadata JSONB,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    CHECK (from_account_id != to_account_id)
);


-- Ledger entries table (immutable) - NO balance column!
CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id),
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    asset_type_id VARCHAR(50) NOT NULL REFERENCES asset_types(id),
    amount BIGINT NOT NULL CHECK (amount != 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- Indexes for performance
CREATE INDEX idx_transactions_idempotency ON transactions(idempotency_key);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_ledger_account_asset ON ledger_entries(account_id, asset_type_id);
CREATE INDEX idx_ledger_transaction ON ledger_entries(transaction_id);


-- Trigger to prevent updates/deletes on ledger_entries
CREATE OR REPLACE FUNCTION prevent_ledger_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Ledger entries are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_ledger_update
    BEFORE UPDATE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_ledger_modification();

CREATE TRIGGER prevent_ledger_delete
    BEFORE DELETE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_ledger_modification();


-- View for current balances
CREATE VIEW account_balances AS
SELECT
    account_id,
    asset_type_id,
    SUM(amount) as balance
FROM ledger_entries
GROUP BY account_id, asset_type_id;


