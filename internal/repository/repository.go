package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wallet-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for data access operations
type Repository interface {
	// Transaction operations
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (*models.Transaction, error)
	GetTransactionByIdempotencyKeyTx(ctx context.Context, tx pgx.Tx, key string) (*models.Transaction, error)
	CreateTransactionIdempotent(ctx context.Context, tx pgx.Tx, transaction *models.Transaction) (int64, error)
	UpdateTransactionStatus(ctx context.Context, tx pgx.Tx, txID int64, status string, completedAt *time.Time) error

	// Ledger operations
	CreateLedgerEntry(ctx context.Context, tx pgx.Tx, entry *models.LedgerEntry) error

	// Balance operations (computed from ledger entries)
	GetBalance(ctx context.Context, accountID int64, assetTypeID string) (int64, error)
	GetBalanceForUpdate(ctx context.Context, tx pgx.Tx, accountID int64, assetTypeID string) (int64, error)
	GetAccountBalances(ctx context.Context, accountID int64) ([]models.Balance, error)

	// Account operations
	GetAccount(ctx context.Context, accountID int64) (*models.Account, error)
	GetAccountForUpdate(ctx context.Context, tx pgx.Tx, accountID int64) (*models.Account, error)

	// Asset type operations
	GetAssetType(ctx context.Context, assetTypeID string) (*models.AssetType, error)

	// Transaction history
	GetTransactionHistory(ctx context.Context, accountID int64, limit, offset int) ([]models.Transaction, error)

	// Transaction management
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

// BeginTx starts a new database transaction with SERIALIZABLE isolation
func (r *PostgresRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
}

// GetAssetType retrieves an asset type by ID
func (r *PostgresRepository) GetAssetType(ctx context.Context, assetTypeID string) (*models.AssetType, error) {
	query := `
		SELECT id, name, description, created_at
		FROM asset_types
		WHERE id = $1
	`

	var assetType models.AssetType
	err := r.pool.QueryRow(ctx, query, assetTypeID).Scan(
		&assetType.ID,
		&assetType.Name,
		&assetType.Description,
		&assetType.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("asset type not found: %s", assetTypeID)
		}
		return nil, fmt.Errorf("failed to get asset type: %w", err)
	}

	return &assetType, nil
}

// GetAccount retrieves an account by ID
func (r *PostgresRepository) GetAccount(ctx context.Context, accountID int64) (*models.Account, error) {
	query := `
		SELECT id, type, owner_id, created_at
		FROM accounts
		WHERE id = $1
	`

	var account models.Account
	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&account.ID,
		&account.Type,
		&account.OwnerID,
		&account.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("account not found: %d", accountID)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// GetAccountForUpdate retrieves an account with FOR UPDATE lock
func (r *PostgresRepository) GetAccountForUpdate(ctx context.Context, tx pgx.Tx, accountID int64) (*models.Account, error) {
	query := `
		SELECT id, type, owner_id, created_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var account models.Account
	err := tx.QueryRow(ctx, query, accountID).Scan(
		&account.ID,
		&account.Type,
		&account.OwnerID,
		&account.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("account not found: %d", accountID)
		}
		return nil, fmt.Errorf("failed to get account for update: %w", err)
	}

	return &account, nil
}

// GetBalance computes balance from ledger entries (NO balance column!)
func (r *PostgresRepository) GetBalance(ctx context.Context, accountID int64, assetTypeID string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0) as balance
		FROM ledger_entries
		WHERE account_id = $1 AND asset_type_id = $2
	`

	var balance int64
	err := r.pool.QueryRow(ctx, query, accountID, assetTypeID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// GetBalanceForUpdate computes balance with account-level locking (CRITICAL FIX)
func (r *PostgresRepository) GetBalanceForUpdate(ctx context.Context, tx pgx.Tx, accountID int64, assetTypeID string) (int64, error) {
	// CRITICAL: Lock the account row first to prevent race conditions
	lockQuery := `SELECT id FROM accounts WHERE id = $1 FOR UPDATE`
	var lockedID int64
	err := tx.QueryRow(ctx, lockQuery, accountID).Scan(&lockedID)
	if err != nil {
		return 0, fmt.Errorf("failed to lock account: %w", err)
	}

	// Now compute the balance (account is already locked)
	balanceQuery := `
		SELECT COALESCE(SUM(amount), 0) as balance
		FROM ledger_entries
		WHERE account_id = $1 AND asset_type_id = $2
	`

	var balance int64
	err = tx.QueryRow(ctx, balanceQuery, accountID, assetTypeID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance for update: %w", err)
	}

	return balance, nil
}

// GetAccountBalances retrieves all balances for an account
func (r *PostgresRepository) GetAccountBalances(ctx context.Context, accountID int64) ([]models.Balance, error) {
	query := `
		SELECT account_id, asset_type_id, SUM(amount) as balance
		FROM ledger_entries
		WHERE account_id = $1
		GROUP BY account_id, asset_type_id
		HAVING SUM(amount) > 0
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balances: %w", err)
	}
	defer rows.Close()

	var balances []models.Balance
	for rows.Next() {
		var balance models.Balance
		err := rows.Scan(&balance.AccountID, &balance.AssetTypeID, &balance.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		balances = append(balances, balance)
	}

	return balances, nil
}

// GetTransactionByIdempotencyKey retrieves transaction by idempotency key
func (r *PostgresRepository) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*models.Transaction, error) {
	query := `
		SELECT id, idempotency_key, type, asset_type_id, amount,
		       from_account_id, to_account_id, metadata, status,
		       created_at, completed_at
		FROM transactions
		WHERE idempotency_key = $1
	`

	var transaction models.Transaction
	var completedAt sql.NullTime

	err := r.pool.QueryRow(ctx, query, key).Scan(
		&transaction.ID,
		&transaction.IdempotencyKey,
		&transaction.Type,
		&transaction.AssetTypeID,
		&transaction.Amount,
		&transaction.FromAccountID,
		&transaction.ToAccountID,
		&transaction.Metadata,
		&transaction.Status,
		&transaction.CreatedAt,
		&completedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found with idempotency key: %s", key)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if completedAt.Valid {
		transaction.CompletedAt = &completedAt.Time
	}

	return &transaction, nil
}

// GetTransactionByIdempotencyKeyTx retrieves transaction within a transaction
func (r *PostgresRepository) GetTransactionByIdempotencyKeyTx(ctx context.Context, tx pgx.Tx, key string) (*models.Transaction, error) {
	query := `
		SELECT id, idempotency_key, type, asset_type_id, amount,
		       from_account_id, to_account_id, metadata, status,
		       created_at, completed_at
		FROM transactions
		WHERE idempotency_key = $1
	`

	var transaction models.Transaction
	var completedAt sql.NullTime

	err := tx.QueryRow(ctx, query, key).Scan(
		&transaction.ID,
		&transaction.IdempotencyKey,
		&transaction.Type,
		&transaction.AssetTypeID,
		&transaction.Amount,
		&transaction.FromAccountID,
		&transaction.ToAccountID,
		&transaction.Metadata,
		&transaction.Status,
		&transaction.CreatedAt,
		&completedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found with idempotency key: %s", key)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if completedAt.Valid {
		transaction.CompletedAt = &completedAt.Time
	}

	return &transaction, nil
}

// CreateTransactionIdempotent creates transaction with idempotency (CRITICAL FIX)
func (r *PostgresRepository) CreateTransactionIdempotent(ctx context.Context, tx pgx.Tx, transaction *models.Transaction) (int64, error) {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(transaction.Metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO transactions (
			idempotency_key, type, asset_type_id, amount,
			from_account_id, to_account_id, metadata, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`

	var id int64
	err = tx.QueryRow(
		ctx, query,
		transaction.IdempotencyKey,
		transaction.Type,
		transaction.AssetTypeID,
		transaction.Amount,
		transaction.FromAccountID,
		transaction.ToAccountID,
		string(metadataJSON),
		transaction.Status,
		transaction.CreatedAt,
	).Scan(&id)

	if err == pgx.ErrNoRows {
		// No rows returned means conflict occurred (idempotent case)
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to create transaction: %w", err)
	}

	transaction.ID = id
	return 1, nil
}

// UpdateTransactionStatus updates transaction status and completion time
func (r *PostgresRepository) UpdateTransactionStatus(ctx context.Context, tx pgx.Tx, txID int64, status string, completedAt *time.Time) error {
	query := `
		UPDATE transactions
		SET status = $2, completed_at = $3
		WHERE id = $1
	`

	_, err := tx.Exec(ctx, query, txID, status, completedAt)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

// CreateLedgerEntry creates a new ledger entry
func (r *PostgresRepository) CreateLedgerEntry(ctx context.Context, tx pgx.Tx, entry *models.LedgerEntry) error {
	query := `
		INSERT INTO ledger_entries (
			transaction_id, account_id, asset_type_id, amount, created_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := tx.QueryRow(
		ctx, query,
		entry.TransactionID,
		entry.AccountID,
		entry.AssetTypeID,
		entry.Amount,
		entry.CreatedAt,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to create ledger entry: %w", err)
	}

	return nil
}

// GetTransactionHistory retrieves transaction history for an account
func (r *PostgresRepository) GetTransactionHistory(ctx context.Context, accountID int64, limit, offset int) ([]models.Transaction, error) {
	query := `
		SELECT id, idempotency_key, type, asset_type_id, amount,
		       from_account_id, to_account_id, metadata, status,
		       created_at, completed_at
		FROM transactions
		WHERE from_account_id = $1 OR to_account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var completedAt sql.NullTime

		err := rows.Scan(
			&transaction.ID,
			&transaction.IdempotencyKey,
			&transaction.Type,
			&transaction.AssetTypeID,
			&transaction.Amount,
			&transaction.FromAccountID,
			&transaction.ToAccountID,
			&transaction.Metadata,
			&transaction.Status,
			&transaction.CreatedAt,
			&completedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		if completedAt.Valid {
			transaction.CompletedAt = &completedAt.Time
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
