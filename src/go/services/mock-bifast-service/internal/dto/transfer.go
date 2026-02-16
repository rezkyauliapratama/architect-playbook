// src/go/services/mock-bifast-service/internal/dto/transfer.go
package dto

import "fmt"

// TransferRequest represents BI-FAST transfer request
type TransferRequest struct {
	// Source information
	SourceBankCode      string `json:"sourceBankCode" validate:"required,len=8"`
	SourceAccountNumber string `json:"sourceAccountNumber" validate:"required,min=1,max=50"`

	// Destination information
	DestBankCode      string `json:"destBankCode" validate:"required,len=8"`
	DestAccountNumber string `json:"destAccountNumber" validate:"required,min=1,max=50"`

	// Transaction details
	Amount      string `json:"amount" validate:"required,numeric"`
	Currency    string `json:"currency" validate:"required,len=3"`
	Description string `json:"description" validate:"max=255"`

	// Reference and idempotency
	ReferenceID    string `json:"referenceId" validate:"required,min=1,max=100"`
	IdempotencyKey string `json:"-"` // Populated from header
}

// Validate validates the transfer request
func (r *TransferRequest) Validate() error {
	// Sanitize inputs
	r.SourceBankCode = SanitizeBankCode(r.SourceBankCode)
	r.SourceAccountNumber = SanitizeAccountNumber(r.SourceAccountNumber)
	r.DestBankCode = SanitizeBankCode(r.DestBankCode)
	r.DestAccountNumber = SanitizeAccountNumber(r.DestAccountNumber)

	// Validate source bank code
	if len(r.SourceBankCode) < 3 || len(r.SourceBankCode) > 8 {
		return fmt.Errorf("source bank code must be 3-8 characters")
	}

	// Validate source account number
	if r.SourceAccountNumber == "" {
		return fmt.Errorf("source account number is required")
	}

	if len(r.SourceAccountNumber) > 50 {
		return fmt.Errorf("source account number must be max 50 characters")
	}

	// Validate destination bank code
	if len(r.DestBankCode) < 3 || len(r.DestBankCode) > 8 {
		return fmt.Errorf("destination bank code must be 3-8 characters")
	}

	// Validate destination account number
	if r.DestAccountNumber == "" {
		return fmt.Errorf("destination account number is required")
	}

	if len(r.DestAccountNumber) > 50 {
		return fmt.Errorf("destination account number must be max 50 characters")
	}

	// Validate amount
	if err := ValidateAmount(r.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	// Validate currency (must be 3 uppercase letters)
	if len(r.Currency) != 3 {
		return fmt.Errorf("currency must be 3 characters (e.g., IDR, USD)")
	}

	// Validate reference ID
	if r.ReferenceID == "" {
		return fmt.Errorf("reference ID is required")
	}

	if len(r.ReferenceID) > 100 {
		return fmt.Errorf("reference ID must be max 100 characters")
	}

	// Validate idempotency key
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency key is required (X-Idempotency-Key header)")
	}

	// Validate description length
	if len(r.Description) > 255 {
		return fmt.Errorf("description must be max 255 characters")
	}

	return nil
}

// TransferResponse represents BI-FAST transfer response
type TransferResponse struct {
	// Common response fields
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp,omitempty"`

	// Response status
	ResponseCode string `json:"responseCode"`
	ResponseMsg  string `json:"responseMsg"`

	// Transaction information
	TransactionID string `json:"transactionId,omitempty"`
	ReferenceID   string `json:"referenceId"`

	// Amount information
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
	Fee      string `json:"fee,omitempty"`

	// Account names
	SourceAccountName string `json:"sourceAccountName,omitempty"`
	DestAccountName   string `json:"destAccountName,omitempty"`

	// Status and timing
	Status          string `json:"status,omitempty"`
	TransactionTime string `json:"transactionTime,omitempty"`
}
