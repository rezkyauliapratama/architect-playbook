// src/go/services/mock-bifast-service/internal/models/account.go
package models

import "time"

// Account represents a mock bank account for BI-FAST testing
type Account struct {
	// Account identification
	BankCode      string `json:"bankCode"`      // Bank code (8 characters, e.g., CENAIDJA)
	AccountNumber string `json:"accountNumber"` // Account number (1-50 alphanumeric)
	AccountName   string `json:"accountName"`   // Account holder name
	AccountType   string `json:"accountType"`   // Account type (savings, checking, etc)

	// Account balance (for testing purposes)
	Balance  string `json:"balance"`  // Current balance (decimal string)
	Currency string `json:"currency"` // Currency code (IDR)

	// Account status
	Status    string `json:"status"`    // Account status (active, blocked, closed)
	IsActive  bool   `json:"isActive"`  // Quick check if account is active
	IsBlocked bool   `json:"isBlocked"` // Quick check if account is blocked

	// Timestamps
	CreatedAt time.Time `json:"createdAt"` // Account creation timestamp
	UpdatedAt time.Time `json:"updatedAt"` // Last update timestamp
}

// AccountStatus constants
const (
	AccountStatusActive  = "active"  // Account is active and can be used
	AccountStatusBlocked = "blocked" // Account is blocked
	AccountStatusClosed  = "closed"  // Account is closed
)
