// src/go/services/mock-bifast-service/internal/dto/validation.go
package dto

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	// Bank code format: 3-8 alphanumeric characters (flexible for Indonesian banks)
	bankCodeRegex = regexp.MustCompile(`^[A-Z0-9]{3,8}$`)

	// Account number format: 1-50 alphanumeric characters
	accountNumberRegex = regexp.MustCompile(`^[A-Z0-9]{1,50}$`)

	// Reference ID format: alphanumeric with hyphens
	referenceIDRegex = regexp.MustCompile(`^[A-Z0-9\-]{1,100}$`)

	// Currency code format: 3 uppercase letters
	currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)
)

// ValidationResult represents validation result
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidateAccountInquiryRequest validates account inquiry request
func ValidateAccountInquiryRequest(req *AccountInquiryRequest) ValidationResult {
	result := ValidationResult{Valid: true, Errors: []ValidationError{}}

	// Validate bank code
	if !bankCodeRegex.MatchString(req.BankCode) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "bankCode",
			Message: "Bank code must be 3-8 uppercase alphanumeric characters",
		})
	}

	// Validate account number
	if !accountNumberRegex.MatchString(req.AccountNumber) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "accountNumber",
			Message: "Account number must be 1-50 alphanumeric characters",
		})
	}

	// Validate reference ID
	if req.ReferenceID == "" || len(req.ReferenceID) > 100 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "referenceId",
			Message: "Reference ID is required and must be max 100 characters",
		})
	}

	return result
}

// ValidateTransferRequest validates transfer request
func ValidateTransferRequest(req *TransferRequest) ValidationResult {
	result := ValidationResult{Valid: true, Errors: []ValidationError{}}

	// Validate source bank code
	if !bankCodeRegex.MatchString(req.SourceBankCode) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "sourceBankCode",
			Message: "Source bank code must be 3-8 uppercase alphanumeric characters",
		})
	}

	// Validate source account number
	if !accountNumberRegex.MatchString(req.SourceAccountNumber) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "sourceAccountNumber",
			Message: "Source account number must be 1-50 alphanumeric characters",
		})
	}

	// Validate destination bank code
	if !bankCodeRegex.MatchString(req.DestBankCode) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "destBankCode",
			Message: "Destination bank code must be 3-8 uppercase alphanumeric characters",
		})
	}

	// Validate destination account number
	if !accountNumberRegex.MatchString(req.DestAccountNumber) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "destAccountNumber",
			Message: "Destination account number must be 1-50 alphanumeric characters",
		})
	}

	// Validate amount
	if err := ValidateAmount(req.Amount); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "amount",
			Message: err.Error(),
		})
	}

	// Validate currency
	if !currencyRegex.MatchString(req.Currency) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "currency",
			Message: "Currency must be 3 uppercase letters (e.g., IDR, USD)",
		})
	}

	// Validate reference ID
	if req.ReferenceID == "" || len(req.ReferenceID) > 100 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "referenceId",
			Message: "Reference ID is required and must be max 100 characters",
		})
	}

	// Validate idempotency key
	if req.IdempotencyKey == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "X-Idempotency-Key",
			Message: "Idempotency key header is required",
		})
	}

	// Validate description length
	if len(req.Description) > 255 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "description",
			Message: "Description must be max 255 characters",
		})
	}

	return result
}

// ValidateAmount validates monetary amount
func ValidateAmount(amount string) error {
	// Remove whitespace
	amount = strings.TrimSpace(amount)

	// Check if empty
	if amount == "" {
		return fmt.Errorf("amount is required")
	}

	// Parse as decimal
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return fmt.Errorf("amount must be a valid decimal number")
	}

	// Check if positive
	if amt.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("amount must be greater than zero")
	}

	// Check maximum amount (e.g., 10 billion IDR)
	maxAmount := decimal.NewFromInt(10_000_000_000)
	if amt.GreaterThan(maxAmount) {
		return fmt.Errorf("amount exceeds maximum allowed (10,000,000,000)")
	}

	// Check decimal places (max 2 for IDR)
	if amt.Exponent() < -2 {
		return fmt.Errorf("amount can have maximum 2 decimal places")
	}

	return nil
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, limit int) (int, int, error) {
	// Default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	// Maximum limit
	if limit > 100 {
		limit = 100
	}

	return page, limit, nil
}

// SanitizeAccountNumber sanitizes account number (uppercase and remove special chars)
func SanitizeAccountNumber(accountNumber string) string {
	return strings.ToUpper(strings.TrimSpace(accountNumber))
}

// SanitizeBankCode sanitizes bank code (uppercase and remove whitespace)
func SanitizeBankCode(bankCode string) string {
	return strings.ToUpper(strings.TrimSpace(bankCode))
}

// FormatAmount formats amount to 2 decimal places
func FormatAmount(amount string) string {
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return "0.00"
	}
	return amt.StringFixed(2)
}

// ParseAmount parses amount string to float64
func ParseAmount(amount string) (float64, error) {
	amt, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format")
	}
	return amt, nil
}
