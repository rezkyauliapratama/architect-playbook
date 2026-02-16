// src/go/services/mock-bifast-service/internal/dto/account_inquiry.go
package dto

import "fmt"

// AccountInquiryRequest represents account inquiry request
type AccountInquiryRequest struct {
	BankCode      string `json:"bankCode" validate:"required,len=8"`
	AccountNumber string `json:"accountNumber" validate:"required,min=1,max=50"`
	ReferenceID   string `json:"referenceId" validate:"required,min=1,max=100"`
}

// Validate validates the account inquiry request
func (r *AccountInquiryRequest) Validate() error {
	// Sanitize inputs
	r.BankCode = SanitizeBankCode(r.BankCode)
	r.AccountNumber = SanitizeAccountNumber(r.AccountNumber)

	// Validate bank code format (3 digits for Indonesian banks)
	if len(r.BankCode) < 3 || len(r.BankCode) > 8 {
		return fmt.Errorf("bank code must be 3-8 characters")
	}

	// Validate account number
	if r.AccountNumber == "" {
		return fmt.Errorf("account number is required")
	}

	if len(r.AccountNumber) > 50 {
		return fmt.Errorf("account number must be max 50 characters")
	}

	// Validate reference ID
	if r.ReferenceID == "" {
		return fmt.Errorf("reference ID is required")
	}

	if len(r.ReferenceID) > 100 {
		return fmt.Errorf("reference ID must be max 100 characters")
	}

	return nil
}

// AccountInquiryResponse represents account inquiry response
type AccountInquiryResponse struct {
	Success       bool   `json:"success"`
	ResponseCode  string `json:"responseCode"`
	ResponseMsg   string `json:"responseMsg"`
	ReferenceID   string `json:"referenceId"`
	BankCode      string `json:"bankCode,omitempty"`
	AccountNumber string `json:"accountNumber,omitempty"`
	AccountName   string `json:"accountName,omitempty"`
	AccountType   string `json:"accountType,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}
