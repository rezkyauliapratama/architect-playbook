package models

import "time"

// TransactionStatus represents transaction status
type TransactionStatus string

const (
	StatusPending    TransactionStatus = "PENDING"
	StatusProcessing TransactionStatus = "PROCESSING"
	StatusCompleted  TransactionStatus = "COMPLETED"
	StatusFailed     TransactionStatus = "FAILED"
	StatusExpired    TransactionStatus = "EXPIRED"
)

// Transaction represents a BI-FAST transaction
type Transaction struct {
	TransactionID       string     `json:"transactionId"`
	ReferenceID         string     `json:"referenceId"`
	IdempotencyKey      string     `json:"idempotencyKey"`
	SourceBankCode      string     `json:"sourceBankCode"`
	SourceAccountNumber string     `json:"sourceAccountNumber"`
	DestBankCode        string     `json:"destBankCode"`
	DestAccountNumber   string     `json:"destAccountNumber"`
	Amount              string     `json:"amount"`
	Currency            string     `json:"currency"`
	Fee                 string     `json:"fee"`
	Description         string     `json:"description"`
	Status              string     `json:"status"`
	ResponseCode        string     `json:"responseCode"`
	ResponseMsg         string     `json:"responseMsg"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
}

// TransactionStatistics represents transaction statistics
type TransactionStatistics struct {
	TotalTransactions int    `json:"totalTransactions"`
	CompletedCount    int    `json:"completedCount"`
	FailedCount       int    `json:"failedCount"`
	PendingCount      int    `json:"pendingCount"`
	TotalAmount       string `json:"totalAmount"`
	TotalFee          string `json:"totalFee"`
}
