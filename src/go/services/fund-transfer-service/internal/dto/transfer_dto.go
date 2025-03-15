// internal/dto/transfer_dto.go
package dto

import (
	"errors"
	"time"
)

type CreateTransferRequest struct {
	SourceAccountID          string  `json:"sourceAccountId"`
	DestinationAccountNumber string  `json:"destinationAccountNumber"`
	DestinationBankCode      string  `json:"destinationBankCode"`
	DestinationAccountName   string  `json:"destinationAccountName"`
	Amount                   float64 `json:"amount"`
	Currency                 string  `json:"currency"`
	Description              string  `json:"description"`
	// Dynamic ledger fields
	FeeType              string   `json:"feeType,omitempty"`
	CustomFee            *float64 `json:"customFee,omitempty"`
	DebitAccountCode     string   `json:"debitAccountCode,omitempty"`
	CreditAccountCode    string   `json:"creditAccountCode,omitempty"`
	FeeDebitAccountCode  string   `json:"feeDebitAccountCode,omitempty"`
	FeeCreditAccountCode string   `json:"feeCreditAccountCode,omitempty"`
	DebitDescription     string   `json:"debitDescription,omitempty"`
	CreditDescription    string   `json:"creditDescription,omitempty"`
	FeeDebitDescription  string   `json:"feeDebitDescription,omitempty"`
	FeeCreditDescription string   `json:"feeCreditDescription,omitempty"`
	ReferenceType        string   `json:"referenceType,omitempty"`
}

func (r *CreateTransferRequest) Validate() error {
	if r.SourceAccountID == "" {
		return errors.New("source account ID is required")
	}
	if r.DestinationAccountNumber == "" {
		return errors.New("destination account number is required")
	}
	if r.DestinationBankCode == "" {
		return errors.New("destination bank code is required")
	}
	if r.DestinationAccountName == "" {
		return errors.New("destination account name is required")
	}
	if r.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if r.Currency == "" {
		return errors.New("currency is required")
	}

	return nil
}

type TransferResponse struct {
	TransferID               string     `json:"transferId"`
	SourceAccountID          string     `json:"sourceAccountId"`
	DestinationAccountNumber string     `json:"destinationAccountNumber"`
	DestinationBankCode      string     `json:"destinationBankCode"`
	DestinationAccountName   string     `json:"destinationAccountName"`
	Amount                   float64    `json:"amount"`
	Currency                 string     `json:"currency"`
	Status                   string     `json:"status"`
	ReferenceNumber          string     `json:"referenceNumber"`
	Fee                      float64    `json:"fee,omitempty"`
	Description              string     `json:"description"`
	CreatedAt                time.Time  `json:"createdAt"`
	CompletedAt              *time.Time `json:"completedAt,omitempty"`
	TransactionIDs           []string   `json:"transactionIds,omitempty"`
	LedgerJournalID          *string    `json:"ledgerJournalId,omitempty"`
}

type CreateBulkTransferRequest struct {
	SourceAccountID string             `json:"sourceAccountId"`
	Transfers       []BulkTransferItem `json:"transfers"`
	Currency        string             `json:"currency"`
	BatchReference  string             `json:"batchReference"`
	// Ledger account code overrides
	DebitAccountCode     string `json:"debitAccountCode,omitempty"`
	CreditAccountCode    string `json:"creditAccountCode,omitempty"`
	FeeDebitAccountCode  string `json:"feeDebitAccountCode,omitempty"`
	FeeCreditAccountCode string `json:"feeCreditAccountCode,omitempty"`

	// Description overrides
	DebitDescription     string `json:"debitDescription,omitempty"`
	CreditDescription    string `json:"creditDescription,omitempty"`
	FeeDebitDescription  string `json:"feeDebitDescription,omitempty"`
	FeeCreditDescription string `json:"feeCreditDescription,omitempty"`

	// Other fields
	ReferenceType string `json:"referenceType,omitempty"`
}

type BulkTransferItem struct {
	DestinationAccountNumber string  `json:"destinationAccountNumber"`
	DestinationBankCode      string  `json:"destinationBankCode"`
	DestinationAccountName   string  `json:"destinationAccountName"`
	Amount                   float64 `json:"amount"`
	Description              string  `json:"description"`
}

type BulkTransferResponse struct {
	BulkTransferID  string    `json:"bulkTransferId"`
	SourceAccountID string    `json:"sourceAccountId"`
	TotalAmount     float64   `json:"totalAmount"`
	TransferCount   int       `json:"transferCount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	BatchReference  string    `json:"batchReference"`
	TransferIDs     []string  `json:"transferIds"`
	CreatedAt       time.Time `json:"createdAt"`
}

// DTOs for other service clients
type AccountResponse struct {
	AccountID     string  `json:"accountId"`
	AccountNumber string  `json:"accountNumber"`
	AccountName   string  `json:"accountName"`
	AccountType   string  `json:"accountType"`
	BankCode      string  `json:"bankCode"`
	BankName      string  `json:"bankName"`
	Currency      string  `json:"currency"`
	Balance       float64 `json:"balance"`
	Status        string  `json:"status"`
}

type UpdateAccountBalanceRequest struct {
	Adjustment  float64 `json:"adjustment"`
	ReferenceID string  `json:"referenceId"`
	Description string  `json:"description"`
}

type UpdateAccountBalanceResponse struct {
	AccountID         string    `json:"accountId"`
	NewBalance        float64   `json:"newBalance"`
	AdjustmentApplied float64   `json:"adjustmentApplied"`
	Timestamp         time.Time `json:"timestamp"`
}

type CreateTransactionRequest struct {
	AccountID   string  `json:"accountId"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	ReferenceID string  `json:"referenceId"`
	Description string  `json:"description"`
}

type TransactionResponse struct {
	TransactionID string    `json:"transactionId"`
	AccountID     string    `json:"accountId"`
	Type          string    `json:"type"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	ReferenceID   string    `json:"referenceId"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"createdAt"`
}

type LedgerEntry struct {
	AccountingCode string  `json:"accountingCode"`
	EntryType      string  `json:"entryType"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	ReferenceType  string  `json:"referenceType"`
	ReferenceID    string  `json:"referenceId"`
	Description    string  `json:"description"`
}

type CreateLedgerEntriesRequest struct {
	Entries []LedgerEntry `json:"entries"`
}

type LedgerEntriesResponse struct {
	JournalID string    `json:"journalId"`
	EntryIDs  []string  `json:"entryIds"`
	CreatedAt time.Time `json:"createdAt"`
}
