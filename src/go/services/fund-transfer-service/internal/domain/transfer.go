// internal/domain/transfer.go
package domain

import (
	"time"
)

type TransferStatus string

const (
	TransferStatusPending    TransferStatus = "PENDING"
	TransferStatusProcessing TransferStatus = "PROCESSING"
	TransferStatusCompleted  TransferStatus = "SELESAI"
	TransferStatusFailed     TransferStatus = "GAGAL"
)

type Transfer struct {
	ID                       string         `json:"id" db:"id"` // UUID stored as string
	TransferID               string         `json:"transferId" db:"transfer_id"`
	SourceAccountID          string         `json:"sourceAccountId" db:"source_account_id"`
	DestinationAccountNumber string         `json:"destinationAccountNumber" db:"destination_account_number"`
	DestinationBankCode      string         `json:"destinationBankCode" db:"destination_bank_code"`
	DestinationAccountName   string         `json:"destinationAccountName" db:"destination_account_name"`
	Amount                   float64        `json:"amount" db:"amount"`
	Currency                 string         `json:"currency" db:"currency"`
	Status                   TransferStatus `json:"status" db:"status"`
	ReferenceNumber          string         `json:"referenceNumber" db:"reference_number"`
	Fee                      float64        `json:"fee" db:"fee"`
	Description              string         `json:"description" db:"description"`
	CreatedAt                time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt                time.Time      `json:"updatedAt" db:"updated_at"`
	CompletedAt              *time.Time     `json:"completedAt,omitempty" db:"completed_at"`
	TransactionIDs           []string       `json:"transactionIds,omitempty" db:"transaction_ids"`
	LedgerJournalID          *string        `json:"ledgerJournalId,omitempty" db:"ledger_journal_id"`
}

type BulkTransfer struct {
	ID              string         `json:"id" db:"id"` // UUID stored as string
	BulkTransferID  string         `json:"bulkTransferId" db:"bulk_transfer_id"`
	SourceAccountID string         `json:"sourceAccountId" db:"source_account_id"`
	TotalAmount     float64        `json:"totalAmount" db:"total_amount"`
	TransferCount   int            `json:"transferCount" db:"transfer_count"`
	Currency        string         `json:"currency" db:"currency"`
	Status          TransferStatus `json:"status" db:"status"`
	BatchReference  string         `json:"batchReference" db:"batch_reference"`
	CreatedAt       time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time      `json:"updatedAt" db:"updated_at"`
	CompletedAt     *time.Time     `json:"completedAt,omitempty" db:"completed_at"`
	TransferIDs     []string       `json:"transferIds,omitempty" db:"-"`
}

type BulkTransferItem struct {
	ID             string         `json:"id" db:"id"` // UUID stored as string
	BulkTransferID string         `json:"bulkTransferId" db:"bulk_transfer_id"`
	TransferID     string         `json:"transferId" db:"transfer_id"`
	Status         TransferStatus `json:"status" db:"status"`
	CreatedAt      time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time      `json:"updatedAt" db:"updated_at"`
}
