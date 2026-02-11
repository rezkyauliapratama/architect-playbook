package models

import "time"

// AccountStatus represents account status
type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "ACTIVE"
	AccountStatusInactive AccountStatus = "INACTIVE"
	AccountStatusBlocked  AccountStatus = "BLOCKED"
)

// AccountType represents account type
type AccountType string

const (
	AccountTypeSavings AccountType = "SAVINGS"
	AccountTypeCurrent AccountType = "CURRENT"
)

// Account represents a bank account
type Account struct {
	BankCode      string    `json:"bankCode"`
	AccountNumber string    `json:"accountNumber"`
	AccountName   string    `json:"accountName"`
	AccountType   string    `json:"accountType"`
	Balance       string    `json:"balance"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
