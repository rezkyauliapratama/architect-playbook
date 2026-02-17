// src/go/services/mock-bifast-service/internal/dto/transfer.go
package dto

import (
	"encoding/json"
	"fmt"
)

// TransferRequest represents BI-FAST transfer request
type TransferRequest struct {
	ReferenceID         string `json:"reference_id" validate:"required,max=50"`
	IdempotencyKey      string `json:"idempotency_key" validate:"required,max=100"`
	SourceBankCode      string `json:"source_bank_code" validate:"required,len=8"`
	SourceAccountNumber string `json:"source_account_number" validate:"required,min=10,max=20"`
	DestBankCode        string `json:"dest_bank_code" validate:"required,len=8"`
	DestAccountNumber   string `json:"dest_account_number" validate:"required,min=10,max=20"`
	Amount              string `json:"amount" validate:"required"`
	Currency            string `json:"currency" validate:"required,len=3"`
	Description         string `json:"description" validate:"max=200"`
}

// UnmarshalJSON custom unmarshaler to handle both number and string for amount
func (r *TransferRequest) UnmarshalJSON(data []byte) error {
	// Create an alias type to avoid recursion
	type Alias TransferRequest

	// First, try to unmarshal normally (when amount is string)
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err == nil {
		// Successfully unmarshaled, amount is already a string
		return nil
	}

	// If failed, amount might be a number, so use raw unmarshaling
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Convert amount from number to string if needed
	if amountVal, ok := raw["amount"]; ok {
		switch v := amountVal.(type) {
		case float64:
			// ✅ Convert number to string with 2 decimal places
			r.Amount = fmt.Sprintf("%.2f", v)
		case int:
			r.Amount = fmt.Sprintf("%d.00", v)
		case string:
			r.Amount = v
		default:
			return fmt.Errorf("invalid amount type: %T", v)
		}
		delete(raw, "amount") // Remove to avoid re-processing
	}

	// Marshal back without amount field, then unmarshal to struct
	remaining, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	// Unmarshal remaining fields
	type TempAlias TransferRequest
	temp := (*TempAlias)(r)
	return json.Unmarshal(remaining, temp)
}

// Validate validates the transfer request
func (r *TransferRequest) Validate() error {
	// Check required fields
	if r.ReferenceID == "" {
		return fmt.Errorf("reference_id is required")
	}
	if r.SourceBankCode == "" {
		return fmt.Errorf("source_bank_code is required")
	}
	if r.SourceAccountNumber == "" {
		return fmt.Errorf("source_account_number is required")
	}
	if r.DestBankCode == "" {
		return fmt.Errorf("dest_bank_code is required")
	}
	if r.DestAccountNumber == "" {
		return fmt.Errorf("dest_account_number is required")
	}
	if r.Amount == "" {
		return fmt.Errorf("amount is required")
	}

	// Validate bank code format (8 characters)
	if len(r.SourceBankCode) != 8 {
		return fmt.Errorf("source_bank_code must be exactly 8 characters")
	}
	if len(r.DestBankCode) != 8 {
		return fmt.Errorf("dest_bank_code must be exactly 8 characters")
	}

	// Validate account number length
	if len(r.SourceAccountNumber) < 10 || len(r.SourceAccountNumber) > 20 {
		return fmt.Errorf("source_account_number must be between 10-20 characters")
	}
	if len(r.DestAccountNumber) < 10 || len(r.DestAccountNumber) > 20 {
		return fmt.Errorf("dest_account_number must be between 10-20 characters")
	}

	// Validate amount format
	if _, err := ParseAmount(r.Amount); err != nil {
		return fmt.Errorf("invalid amount format: %w", err)
	}

	// Set default currency if not provided
	if r.Currency == "" {
		r.Currency = "IDR"
	}

	// Validate currency (must be 3 characters)
	if len(r.Currency) != 3 {
		return fmt.Errorf("currency must be exactly 3 characters")
	}

	// Validate description length
	if len(r.Description) > 200 {
		return fmt.Errorf("description must not exceed 200 characters")
	}

	return nil
}

// TransferResponse represents BI-FAST transfer response
type TransferResponse struct {
	Success           bool   `json:"success"`
	ResponseCode      string `json:"response_code"`
	ResponseMsg       string `json:"response_message"`
	TransactionID     string `json:"transaction_id,omitempty"`
	ReferenceID       string `json:"reference_id"`
	Amount            string `json:"amount,omitempty"`
	Currency          string `json:"currency,omitempty"`
	Fee               string `json:"fee,omitempty"`
	SourceAccountName string `json:"source_account_name,omitempty"`
	DestAccountName   string `json:"dest_account_name,omitempty"`
	Status            string `json:"status,omitempty"`
	TransactionTime   string `json:"transaction_time,omitempty"`
	Timestamp         string `json:"timestamp"`
}
