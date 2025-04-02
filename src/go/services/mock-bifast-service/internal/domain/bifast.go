// internal/domain/bifast.go
package domain

import (
	"time"
)

type ProxyType string
type TransactionStatus string

const (
	ProxyTypeEmail ProxyType = "EMAIL"
	ProxyTypePhone ProxyType = "PHONE"

	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// BankAccount represents a bank account for inquiry
// BankAccount represents a bank account for inquiry
type BankAccount struct {
	ID            string    `json:"id" db:"id"`
	AccountNumber string    `json:"accountNumber" db:"account_number"`
	AccountName   string    `json:"accountName" db:"account_name"`
	BankCode      string    `json:"bankCode" db:"bank_code"`
	BankName      string    `json:"bankName" db:"bank_name"`
	ProxyType     *string   `json:"proxyType,omitempty" db:"proxy_type"`
	ProxyValue    *string   `json:"proxyValue,omitempty" db:"proxy_value"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

// Helper method to safely get proxy type value
func (b *BankAccount) GetProxyType() string {
	if b.ProxyType == nil {
		return ""
	}
	return *b.ProxyType
}

// Helper method to safely get proxy value
func (b *BankAccount) GetProxyValue() string {
	if b.ProxyValue == nil {
		return ""
	}
	return *b.ProxyValue
}

// BifastTransaction represents a BI-Fast transfer transaction
type BifastTransaction struct {
	ID                       string            `json:"id" db:"id"`
	TransactionID            string            `json:"transactionId" db:"transaction_id"`
	SourceAccountNumber      string            `json:"sourceAccountNumber" db:"source_account_number"`
	SourceAccountName        string            `json:"sourceAccountName" db:"source_account_name"`
	SourceBankCode           string            `json:"sourceBankCode" db:"source_bank_code"`
	DestinationAccountNumber string            `json:"destinationAccountNumber" db:"destination_account_number"`
	DestinationAccountName   string            `json:"destinationAccountName" db:"destination_account_name"`
	DestinationBankCode      string            `json:"destinationBankCode" db:"destination_bank_code"`
	Amount                   float64           `json:"amount" db:"amount"`
	Fee                      float64           `json:"fee" db:"fee"`
	Currency                 string            `json:"currency" db:"currency"`
	Status                   TransactionStatus `json:"status" db:"status"`
	ReferenceID              string            `json:"referenceId" db:"reference_id"`
	Description              string            `json:"description" db:"description"`
	CreatedAt                time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt                time.Time         `json:"updatedAt" db:"updated_at"`
	CompletedAt              *time.Time        `json:"completedAt,omitempty" db:"completed_at"`
}
