package models

import (
	"time"
)

// AssetType represents a type of credit or currency
type AssetType struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Account represents a user or system account
type Account struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`     // "user" or "system"
	OwnerID   string    `json:"owner_id"` // User ID or system identifier
	CreatedAt time.Time `json:"created_at"`
}

// Transaction represents a wallet transaction
type Transaction struct {
	ID             int64      `json:"id"`
	IdempotencyKey string     `json:"idempotency_key"`
	Type           string     `json:"type"` // "topup", "bonus", "spend"
	AssetTypeID    string     `json:"asset_type_id"`
	Amount         int64      `json:"amount"`
	FromAccountID  int64      `json:"from_account_id"`
	ToAccountID    int64      `json:"to_account_id"`
	Metadata       string     `json:"metadata"` // JSON metadata
	Status         string     `json:"status"`   // "pending", "completed", "failed"
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// LedgerEntry represents an entry in the double-entry ledger
// NOTE: No Balance field - balances are computed from SUM(amount)
type LedgerEntry struct {
	ID            int64     `json:"id"`
	TransactionID int64     `json:"transaction_id"`
	AccountID     int64     `json:"account_id"`
	AssetTypeID   string    `json:"asset_type_id"`
	Amount        int64     `json:"amount"` // Positive for credit, negative for debit
	CreatedAt     time.Time `json:"created_at"`
}

// Balance represents a computed account balance
type Balance struct {
	AccountID   int64  `json:"account_id"`
	AssetTypeID string `json:"asset_type_id"`
	Amount      int64  `json:"amount"`
}

// TransactionRequest represents a request to execute a transaction
type TransactionRequest struct {
	IdempotencyKey string                 `json:"-"` // From header
	Type           string                 `json:"type"`
	AssetTypeID    string                 `json:"asset_type_id"`
	Amount         int64                  `json:"amount"`
	FromAccountID  int64                  `json:"from_account_id"`
	ToAccountID    int64                  `json:"to_account_id"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// TransactionResponse represents a successful transaction response
type TransactionResponse struct {
	TransactionID int64      `json:"transaction_id"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Idempotent    bool       `json:"idempotent,omitempty"`
}

// BalanceResponse represents a balance query response
type BalanceResponse struct {
	AccountID   int64  `json:"account_id"`
	AssetTypeID string `json:"asset_type_id"`
	Balance     int64  `json:"balance"`
}

// AccountBalancesResponse represents all balances for an account
type AccountBalancesResponse struct {
	AccountID int64     `json:"account_id"`
	Balances  []Balance `json:"balances"`
}

// TransactionHistoryResponse represents transaction history
type TransactionHistoryResponse struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}
