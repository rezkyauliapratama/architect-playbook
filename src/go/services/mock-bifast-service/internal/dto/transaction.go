// src/go/services/mock-bifast-service/internal/dto/transaction.go
package dto

import "github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"

// TransactionStatusResponse represents transaction status query response
type TransactionStatusResponse struct {
	// Common response fields
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp,omitempty"`

	// Response status
	ResponseCode string `json:"responseCode"`
	ResponseMsg  string `json:"responseMsg"`

	// Transaction identification
	TransactionID string `json:"transactionId,omitempty"`
	ReferenceID   string `json:"referenceId,omitempty"`

	// Source information
	SourceBankCode      string `json:"sourceBankCode,omitempty"`
	SourceAccountNumber string `json:"sourceAccountNumber,omitempty"`

	// Destination information
	DestBankCode      string `json:"destBankCode,omitempty"`
	DestAccountNumber string `json:"destAccountNumber,omitempty"`

	// Transaction details
	Amount      string `json:"amount,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Fee         string `json:"fee,omitempty"`
	Description string `json:"description,omitempty"`

	// Status and timing
	Status      string  `json:"status,omitempty"`
	CreatedAt   string  `json:"createdAt,omitempty"`
	UpdatedAt   string  `json:"updatedAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
}

// TransactionItem represents a single transaction in list (for admin endpoints)
type TransactionItem struct {
	TransactionID       string  `json:"transactionId"`
	ReferenceID         string  `json:"referenceId"`
	SourceBankCode      string  `json:"sourceBankCode"`
	SourceAccountNumber string  `json:"sourceAccountNumber"`
	DestBankCode        string  `json:"destBankCode"`
	DestAccountNumber   string  `json:"destAccountNumber"`
	Amount              string  `json:"amount"`
	Currency            string  `json:"currency"`
	Fee                 string  `json:"fee"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
	CompletedAt         *string `json:"completedAt,omitempty"`
}

// ConvertFromModel converts models.Transaction to TransactionItem
func ConvertFromModel(txn *models.Transaction) *TransactionItem {
	item := &TransactionItem{
		TransactionID:       txn.TransactionID,
		ReferenceID:         txn.ReferenceID,
		SourceBankCode:      txn.SourceBankCode,
		SourceAccountNumber: txn.SourceAccountNumber,
		DestBankCode:        txn.DestBankCode,
		DestAccountNumber:   txn.DestAccountNumber,
		Amount:              txn.Amount,
		Currency:            txn.Currency,
		Fee:                 txn.Fee,
		Status:              txn.Status,
		CreatedAt:           txn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           txn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if txn.CompletedAt != nil {
		completedAtStr := txn.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		item.CompletedAt = &completedAtStr
	}

	return item
}

// TransactionListResponse represents paginated transaction list
type TransactionListResponse struct {
	Success    bool               `json:"success"`
	Data       []*TransactionItem `json:"data"`
	Pagination PaginationMeta     `json:"pagination"`
	Timestamp  string             `json:"timestamp"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}
