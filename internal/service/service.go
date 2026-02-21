package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"wallet-service/internal/models"
	"wallet-service/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

// Custom errors
var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrAccountNotFound     = errors.New("account not found")
	ErrAssetTypeNotFound   = errors.New("asset type not found")
	ErrInvalidRequest      = errors.New("invalid request")
)

// TransactionService defines the interface for transaction operations
type TransactionService interface {
	ExecuteTransaction(ctx context.Context, req *models.TransactionRequest) (*models.Transaction, error)
	GetBalance(ctx context.Context, accountID int64, assetTypeID string) (*models.Balance, error)
	GetAccountBalances(ctx context.Context, accountID int64) ([]models.Balance, error)
	GetTransactionHistory(ctx context.Context, accountID int64, limit, offset int) ([]models.Transaction, error)
}

// TransactionServiceImpl implements TransactionService
type TransactionServiceImpl struct {
	repo repository.Repository
}

// NewTransactionService creates a new transaction service
func NewTransactionService(repo repository.Repository) *TransactionServiceImpl {
	return &TransactionServiceImpl{
		repo: repo,
	}
}

// ExecuteTransaction processes a transaction with production-grade patterns
func (s *TransactionServiceImpl) ExecuteTransaction(ctx context.Context, req *models.TransactionRequest) (*models.Transaction, error) {
	// Step 1: Validate request BEFORE starting transaction
	if err := s.validateTransactionRequest(req); err != nil {
		return nil, err
	}

	// Step 2: Execute with retry logic (idempotency check happens inside transaction)
	var tx *models.Transaction
	err := s.retryWithBackoff(ctx, func() error {
		var txErr error
		tx, txErr = s.executeTransactionWithLocking(ctx, req)
		return txErr
	})

	return tx, err
}

// executeTransactionWithLocking implements the corrected transaction flow
func (s *TransactionServiceImpl) executeTransactionWithLocking(ctx context.Context, req *models.TransactionRequest) (*models.Transaction, error) {
	// Begin database transaction with SERIALIZABLE isolation
	dbTx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback(ctx)

	// Step 1: Validate that asset type exists
	if _, err := s.repo.GetAssetType(ctx, req.AssetTypeID); err != nil {
		return nil, fmt.Errorf("asset type validation failed: %w", err)
	}

	// Step 2: Validate accounts exist and acquire locks in deterministic order
	accountIDs := []int64{req.FromAccountID, req.ToAccountID}
	sort.Slice(accountIDs, func(i, j int) bool {
		return accountIDs[i] < accountIDs[j]
	})

	for _, accountID := range accountIDs {
		if _, err := s.repo.GetAccountForUpdate(ctx, dbTx, accountID); err != nil {
			return nil, fmt.Errorf("account validation failed for account %d: %w", accountID, err)
		}
	}

	// Step 3: Check idempotency inside transaction using INSERT ... ON CONFLICT
	metadataJSON, _ := json.Marshal(req.Metadata)
	tx := &models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Type:           req.Type,
		AssetTypeID:    req.AssetTypeID,
		Amount:         req.Amount,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Metadata:       string(metadataJSON),
		Status:         "pending", // CRITICAL FIX: Insert as pending, update to completed after ledger entries
		CreatedAt:      time.Now(),
	}

	// Try to insert transaction with ON CONFLICT DO NOTHING
	rowsAffected, err := s.repo.CreateTransactionIdempotent(ctx, dbTx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// If no rows affected, transaction already exists (idempotent case)
	if rowsAffected == 0 {
		// Read existing transaction within the same transaction for consistency
		existingTx, err := s.repo.GetTransactionByIdempotencyKeyTx(ctx, dbTx, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing transaction: %w", err)
		}
		// Commit to release locks, then return existing transaction
		if err := dbTx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit idempotent transaction: %w", err)
		}
		return existingTx, nil
	}

	// Step 4: Check balances
	fromBalance, err := s.repo.GetBalanceForUpdate(ctx, dbTx, req.FromAccountID, req.AssetTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if fromBalance < req.Amount {
		return nil, ErrInsufficientBalance
	}

	// Step 5: Create ledger entries (double-entry)
	// Debit from source account
	debitEntry := &models.LedgerEntry{
		TransactionID: tx.ID,
		AccountID:     req.FromAccountID,
		AssetTypeID:   req.AssetTypeID,
		Amount:        -req.Amount,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateLedgerEntry(ctx, dbTx, debitEntry); err != nil {
		return nil, fmt.Errorf("failed to create debit entry: %w", err)
	}

	// Credit to destination account
	creditEntry := &models.LedgerEntry{
		TransactionID: tx.ID,
		AccountID:     req.ToAccountID,
		AssetTypeID:   req.AssetTypeID,
		Amount:        req.Amount,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateLedgerEntry(ctx, dbTx, creditEntry); err != nil {
		return nil, fmt.Errorf("failed to create credit entry: %w", err)
	}

	// Step 6: Mark transaction as completed (CRITICAL: Only after ledger entries exist)
	now := time.Now()
	tx.CompletedAt = &now
	tx.Status = "completed"

	if err := s.repo.UpdateTransactionStatus(ctx, dbTx, tx.ID, "completed", &now); err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Step 7: Commit transaction
	if err := dbTx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tx, nil
}

// validateTransactionRequest validates the transaction request
func (s *TransactionServiceImpl) validateTransactionRequest(req *models.TransactionRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("%w: amount must be positive", ErrInvalidRequest)
	}

	if req.FromAccountID == req.ToAccountID {
		return fmt.Errorf("%w: cannot transfer to same account", ErrInvalidRequest)
	}

	if req.AssetTypeID == "" {
		return fmt.Errorf("%w: asset type ID is required", ErrInvalidRequest)
	}

	if req.Type != "topup" && req.Type != "bonus" && req.Type != "spend" {
		return fmt.Errorf("%w: invalid transaction type", ErrInvalidRequest)
	}

	if req.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidRequest)
	}

	return nil
}

// retryWithBackoff implements retry logic with exponential backoff
func (s *TransactionServiceImpl) retryWithBackoff(ctx context.Context, operation func() error) error {
	maxRetries := 3
	baseDelay := 50 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()

		if err == nil {
			return nil
		}

		// Check if error is retryable (serialization failure or deadlock)
		if !isRetryableError(err) {
			return err
		}

		if attempt == maxRetries {
			return fmt.Errorf("max retries exceeded: %w", err)
		}

		// Exponential backoff with jitter
		delay := baseDelay * time.Duration(1<<uint(attempt))
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))

		select {
		case <-time.After(delay + jitter):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	// Check for PostgreSQL serialization errors or deadlocks
	if pgErr, ok := err.(*pgconn.PgError); ok {
		// 40001: serialization_failure
		// 40P01: deadlock_detected
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}

// GetBalance retrieves current balance for an account and asset type
func (s *TransactionServiceImpl) GetBalance(ctx context.Context, accountID int64, assetTypeID string) (*models.Balance, error) {
	// Validate account exists
	if _, err := s.repo.GetAccount(ctx, accountID); err != nil {
		return nil, fmt.Errorf("account validation failed: %w", err)
	}

	// Validate asset type exists
	if _, err := s.repo.GetAssetType(ctx, assetTypeID); err != nil {
		return nil, fmt.Errorf("asset type validation failed: %w", err)
	}

	balance, err := s.repo.GetBalance(ctx, accountID, assetTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return &models.Balance{
		AccountID:   accountID,
		AssetTypeID: assetTypeID,
		Amount:      balance,
	}, nil
}

// GetAccountBalances retrieves all balances for an account
func (s *TransactionServiceImpl) GetAccountBalances(ctx context.Context, accountID int64) ([]models.Balance, error) {
	// Validate account exists
	if _, err := s.repo.GetAccount(ctx, accountID); err != nil {
		return nil, fmt.Errorf("account validation failed: %w", err)
	}

	balances, err := s.repo.GetAccountBalances(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balances: %w", err)
	}

	return balances, nil
}

// GetTransactionHistory retrieves transaction history for an account
func (s *TransactionServiceImpl) GetTransactionHistory(ctx context.Context, accountID int64, limit, offset int) ([]models.Transaction, error) {
	return s.repo.GetTransactionHistory(ctx, accountID, limit, offset)
}
