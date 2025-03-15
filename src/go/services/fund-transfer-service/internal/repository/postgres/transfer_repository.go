// internal/repository/postgres/transfer_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"fund-transfer-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type transferRepository struct {
	db *sqlx.DB
}

func NewTransferRepository(db *sqlx.DB) *transferRepository {
	return &transferRepository{db: db}
}

func (r *transferRepository) Create(ctx context.Context, transfer *domain.Transfer) error {
	// Generate UUID at code level
	transfer.ID = uuid.New().String()

	query := `
        INSERT INTO transfers (
            id, transfer_id, source_account_id, destination_account_number, 
            destination_bank_code, destination_account_name, amount, 
            currency, status, reference_number, fee, description, 
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
        )
    `

	now := time.Now()
	transfer.CreatedAt = now
	transfer.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx, query,
		transfer.ID, transfer.TransferID, transfer.SourceAccountID, transfer.DestinationAccountNumber,
		transfer.DestinationBankCode, transfer.DestinationAccountName, transfer.Amount,
		transfer.Currency, transfer.Status, transfer.ReferenceNumber, transfer.Fee,
		transfer.Description, transfer.CreatedAt, transfer.UpdatedAt,
	)

	return err
}

func (r *transferRepository) GetByID(ctx context.Context, transferID string) (*domain.Transfer, error) {
	query := `SELECT * FROM transfers WHERE transfer_id = $1`

	var transfer domain.Transfer
	var transactionIDsJSON sql.NullString

	err := r.db.GetContext(ctx, &transfer, query, transferID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle transaction IDs stored as JSON
	if transactionIDsJSON.Valid {
		var ids []string
		if err := json.Unmarshal([]byte(transactionIDsJSON.String), &ids); err != nil {
			return nil, err
		}
		transfer.TransactionIDs = ids
	}

	return &transfer, nil
}

func (r *transferRepository) GetBySourceAccountID(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error) {
	query := `
        SELECT * FROM transfers 
        WHERE source_account_id = $1 
        ORDER BY created_at DESC 
        LIMIT $2 OFFSET $3
    `

	rows, err := r.db.QueryxContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []*domain.Transfer
	for rows.Next() {
		var transfer domain.Transfer
		var transactionIDsJSON sql.NullString

		if err := rows.StructScan(&transfer); err != nil {
			return nil, err
		}

		// Handle transaction IDs stored as JSON
		if transactionIDsJSON.Valid {
			var ids []string
			if err := json.Unmarshal([]byte(transactionIDsJSON.String), &ids); err != nil {
				return nil, err
			}
			transfer.TransactionIDs = ids
		}

		transfers = append(transfers, &transfer)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transfers, nil
}

func (r *transferRepository) UpdateStatus(ctx context.Context, transferID string, status domain.TransferStatus) error {
	query := `
        UPDATE transfers 
        SET status = $1, updated_at = $2 
        WHERE transfer_id = $3
    `

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), transferID)
	return err
}

func (r *transferRepository) UpdateTransactionIDs(ctx context.Context, transferID string, transactionIDs []string) error {
	transactionIDsJSON, err := json.Marshal(transactionIDs)
	if err != nil {
		return err
	}

	query := `
        UPDATE transfers 
        SET transaction_ids = $1, updated_at = $2 
        WHERE transfer_id = $3
    `

	_, err = r.db.ExecContext(ctx, query, transactionIDsJSON, time.Now(), transferID)
	return err
}

func (r *transferRepository) UpdateLedgerJournalID(ctx context.Context, transferID string, journalID string) error {
	query := `
        UPDATE transfers 
        SET ledger_journal_id = $1, updated_at = $2 
        WHERE transfer_id = $3
    `

	_, err := r.db.ExecContext(ctx, query, journalID, time.Now(), transferID)
	return err
}

func (r *transferRepository) CompleteTransfer(ctx context.Context, transferID string) error {
	now := time.Now()
	query := `
        UPDATE transfers 
        SET status = $1, updated_at = $2, completed_at = $3 
        WHERE transfer_id = $4
    `

	_, err := r.db.ExecContext(ctx, query, domain.TransferStatusCompleted, now, now, transferID)
	return err
}

func (r *transferRepository) CreateBulkTransfer(ctx context.Context, bulkTransfer *domain.BulkTransfer) error {
	// Generate UUID at code level
	bulkTransfer.ID = uuid.New().String()

	query := `
        INSERT INTO bulk_transfers (
            id, bulk_transfer_id, source_account_id, total_amount, 
            transfer_count, currency, status, batch_reference, 
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
        )
    `

	now := time.Now()
	bulkTransfer.CreatedAt = now
	bulkTransfer.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx, query,
		bulkTransfer.ID, bulkTransfer.BulkTransferID, bulkTransfer.SourceAccountID,
		bulkTransfer.TotalAmount, bulkTransfer.TransferCount, bulkTransfer.Currency,
		bulkTransfer.Status, bulkTransfer.BatchReference, bulkTransfer.CreatedAt,
		bulkTransfer.UpdatedAt,
	)

	return err
}

func (r *transferRepository) AddBulkTransferItem(ctx context.Context, item *domain.BulkTransferItem) error {
	// Generate UUID at code level
	item.ID = uuid.New().String()

	query := `
        INSERT INTO bulk_transfer_items (
            id, bulk_transfer_id, transfer_id, status, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6
        )
    `

	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.ExecContext(
		ctx, query,
		item.ID, item.BulkTransferID, item.TransferID, item.Status, item.CreatedAt, item.UpdatedAt,
	)

	return err
}

func (r *transferRepository) GetBulkTransferByID(ctx context.Context, bulkTransferID string) (*domain.BulkTransfer, error) {
	query := `SELECT * FROM bulk_transfers WHERE bulk_transfer_id = $1`

	var bulkTransfer domain.BulkTransfer
	err := r.db.GetContext(ctx, &bulkTransfer, query, bulkTransferID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &bulkTransfer, nil
}

func (r *transferRepository) GetBulkTransferItems(ctx context.Context, bulkTransferID string) ([]*domain.BulkTransferItem, error) {
	query := `
        SELECT * FROM bulk_transfer_items 
        WHERE bulk_transfer_id = $1 
        ORDER BY created_at ASC
    `

	var items []*domain.BulkTransferItem
	err := r.db.SelectContext(ctx, &items, query, bulkTransferID)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *transferRepository) UpdateBulkTransferStatus(ctx context.Context, bulkTransferID string, status domain.TransferStatus) error {
	query := `
        UPDATE bulk_transfers 
        SET status = $1, updated_at = $2 
        WHERE bulk_transfer_id = $3
    `

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), bulkTransferID)
	return err
}
