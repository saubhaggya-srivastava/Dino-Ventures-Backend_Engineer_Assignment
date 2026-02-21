package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"wallet-service/internal/models"
	"wallet-service/internal/service"

	"github.com/gorilla/mux"
)

// Handler handles HTTP requests for the wallet service
type Handler struct {
	transactionService service.TransactionService
}

// NewHandler creates a new HTTP handler
func NewHandler(transactionService service.TransactionService) *Handler {
	return &Handler{
		transactionService: transactionService,
	}
}

// SetupRoutes configures the HTTP routes
func (h *Handler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(h.corsMiddleware)

	// API routes (must come before static files)
	api := router.PathPrefix("/api/v1").Subrouter()

	// Transaction endpoints
	api.HandleFunc("/transactions", h.ExecuteTransaction).Methods("POST")

	// Balance endpoints
	api.HandleFunc("/accounts/{account_id}/balances/{asset_type_id}", h.GetBalance).Methods("GET")
	api.HandleFunc("/accounts/{account_id}/balances", h.GetAccountBalances).Methods("GET")

	// Transaction history
	api.HandleFunc("/accounts/{account_id}/transactions", h.GetTransactionHistory).Methods("GET")

	// Health check
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")

	// Serve static files (must come last)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	return router
}

// ExecuteTransaction handles POST /api/v1/transactions
func (h *Handler) ExecuteTransaction(w http.ResponseWriter, r *http.Request) {
	// Extract idempotency key from header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key header is required")
		return
	}

	// Parse request body
	var req models.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
		return
	}

	// Set idempotency key
	req.IdempotencyKey = idempotencyKey

	// Execute transaction
	transaction, err := h.transactionService.ExecuteTransaction(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Determine if this was an idempotent request
	isIdempotent := transaction.CreatedAt.Before(time.Now().Add(-1 * time.Second))

	// Create response
	response := models.TransactionResponse{
		TransactionID: transaction.ID,
		Status:        transaction.Status,
		CreatedAt:     transaction.CreatedAt,
		CompletedAt:   transaction.CompletedAt,
		Idempotent:    isIdempotent,
	}

	// Return appropriate status code
	statusCode := http.StatusCreated
	if isIdempotent {
		statusCode = http.StatusOK
	}

	h.writeJSONResponse(w, statusCode, response)
}

// GetBalance handles GET /api/v1/accounts/{account_id}/balances/{asset_type_id}
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	accountID, err := strconv.ParseInt(vars["account_id"], 10, 64)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_account_id", "Invalid account ID")
		return
	}

	assetTypeID := vars["asset_type_id"]
	if assetTypeID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "missing_asset_type_id", "Asset type ID is required")
		return
	}

	balance, err := h.transactionService.GetBalance(r.Context(), accountID, assetTypeID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	response := models.BalanceResponse{
		AccountID:   balance.AccountID,
		AssetTypeID: balance.AssetTypeID,
		Balance:     balance.Amount,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetAccountBalances handles GET /api/v1/accounts/{account_id}/balances
func (h *Handler) GetAccountBalances(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	accountID, err := strconv.ParseInt(vars["account_id"], 10, 64)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_account_id", "Invalid account ID")
		return
	}

	balances, err := h.transactionService.GetAccountBalances(r.Context(), accountID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	response := models.AccountBalancesResponse{
		AccountID: accountID,
		Balances:  balances,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetTransactionHistory handles GET /api/v1/accounts/{account_id}/transactions
func (h *Handler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	accountID, err := strconv.ParseInt(vars["account_id"], 10, 64)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_account_id", "Invalid account ID")
		return
	}

	// Parse query parameters
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	transactions, err := h.transactionService.GetTransactionHistory(r.Context(), accountID, limit, offset)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	response := models.TransactionHistoryResponse{
		Transactions: transactions,
		Total:        len(transactions), // This would be a separate count query in production
		Limit:        limit,
		Offset:       offset,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":  "healthy",
		"service": "wallet-service",
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// handleServiceError converts service errors to appropriate HTTP responses
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInsufficientBalance):
		h.writeErrorResponse(w, http.StatusBadRequest, "insufficient_balance", err.Error())
	case errors.Is(err, service.ErrAccountNotFound):
		h.writeErrorResponse(w, http.StatusNotFound, "account_not_found", err.Error())
	case errors.Is(err, service.ErrAssetTypeNotFound):
		h.writeErrorResponse(w, http.StatusNotFound, "asset_type_not_found", err.Error())
	case errors.Is(err, service.ErrInvalidRequest):
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "An internal error occurred")
	}
}

// writeJSONResponse writes a JSON response
func (h *Handler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error in production
		fmt.Printf("Failed to encode JSON response: %v\n", err)
	}
}

// writeErrorResponse writes an error response
func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	response := models.ErrorResponse{
		Error:   errorCode,
		Message: message,
	}
	h.writeJSONResponse(w, statusCode, response)
}

// corsMiddleware adds CORS headers to all responses
func (h *Handler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
