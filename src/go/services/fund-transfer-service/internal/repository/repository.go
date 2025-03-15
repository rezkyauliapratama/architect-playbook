// internal/repository/repository.go
package repository

import (
	"context"

	"fund-transfer-service/internal/domain"
)

type TransferRepository interface {
	Create(ctx context.Context, transfer *domain.Transfer) error
	GetByID(ctx context.Context, transferID string) (*domain.Transfer, error)
	GetBySourceAccountID(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error)
	UpdateStatus(ctx context.Context, transferID string, status domain.TransferStatus) error
	UpdateTransactionIDs(ctx context.Context, transferID string, transactionIDs []string) error
	UpdateLedgerJournalID(ctx context.Context, transferID string, journalID string) error
	CompleteTransfer(ctx context.Context, transferID string) error

	CreateBulkTransfer(ctx context.Context, bulkTransfer *domain.BulkTransfer) error
	AddBulkTransferItem(ctx context.Context, item *domain.BulkTransferItem) error
	GetBulkTransferByID(ctx context.Context, bulkTransferID string) (*domain.BulkTransfer, error)
	GetBulkTransferItems(ctx context.Context, bulkTransferID string) ([]*domain.BulkTransferItem, error)
	UpdateBulkTransferStatus(ctx context.Context, bulkTransferID string, status domain.TransferStatus) error
}
