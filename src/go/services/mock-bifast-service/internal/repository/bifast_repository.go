// src/go/services/mock-bifast-service/internal/repository/bifast_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"
)

// TransactionRepository defines transaction data access methods
type TransactionRepository interface {
	Create(ctx context.Context, txn *models.Transaction) error
	UpdateStatus(ctx context.Context, transactionID string, status models.TransactionStatus, responseCode, responseMsg string, completedAt *time.Time) error
	FindByID(ctx context.Context, transactionID string) (*models.Transaction, error)
	FindByReferenceID(ctx context.Context, referenceID string) (*models.Transaction, error)
	FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.Transaction, error)
	FindAll(ctx context.Context, limit, offset int) ([]*models.Transaction, int, error)
	GetStatistics(ctx context.Context) (*models.TransactionStatistics, error)
	Delete(ctx context.Context, transactionID string) error
	DeleteAll(ctx context.Context) error
}

type transactionRepository struct {
	db     *pgxpool.Pool
	logger *logger.Logger // ✅ Use libs/logger instead of zerolog
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *pgxpool.Pool, log *logger.Logger) TransactionRepository {
	return &transactionRepository{
		db:     db,
		logger: log,
	}
}

// Create creates a new transaction
func (r *transactionRepository) Create(ctx context.Context, txn *models.Transaction) error {
	query := `
		INSERT INTO transactions (
			transaction_id, reference_id, idempotency_key,
			source_bank_code, source_account_number,
			dest_bank_code, dest_account_number,
			amount, currency, fee, description,
			status, response_code, response_msg,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	_, err := r.db.Exec(ctx, query,
		txn.TransactionID,
		txn.ReferenceID,
		txn.IdempotencyKey,
		txn.SourceBankCode,
		txn.SourceAccountNumber,
		txn.DestBankCode,
		txn.DestAccountNumber,
		txn.Amount,
		txn.Currency,
		txn.Fee,
		txn.Description,
		txn.Status,
		txn.ResponseCode,
		txn.ResponseMsg,
		txn.CreatedAt,
		txn.UpdatedAt,
	)

	if err != nil {
		// ✅ Use ErrorContext instead of zerolog methods
		r.logger.ErrorContext("Failed to create transaction", err, map[string]interface{}{
			"transactionId": txn.TransactionID,
		})
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// ✅ Use InfoContext for success logs
	r.logger.InfoContext("Transaction created successfully", map[string]interface{}{
		"transactionId": txn.TransactionID,
		"amount":        txn.Amount,
		"status":        txn.Status,
	})

	return nil
}

// UpdateStatus updates transaction status
func (r *transactionRepository) UpdateStatus(ctx context.Context, transactionID string, status models.TransactionStatus, responseCode, responseMsg string, completedAt *time.Time) error {
	query := `
		UPDATE transactions
		SET status = $1,
		    response_code = $2,
		    response_msg = $3,
		    completed_at = $4,
		    updated_at = $5
		WHERE transaction_id = $6
	`

	_, err := r.db.Exec(ctx, query,
		string(status),
		responseCode,
		responseMsg,
		completedAt,
		time.Now(),
		transactionID,
	)

	if err != nil {
		r.logger.ErrorContext("Failed to update transaction status", err, map[string]interface{}{
			"transactionId": transactionID,
		})
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	r.logger.InfoContext("Transaction status updated", map[string]interface{}{
		"transactionId": transactionID,
		"status":        string(status),
	})

	return nil
}

// FindByID retrieves a transaction by ID
func (r *transactionRepository) FindByID(ctx context.Context, transactionID string) (*models.Transaction, error) {
	query := `
		SELECT transaction_id, reference_id, idempotency_key,
		       source_bank_code, source_account_number,
		       dest_bank_code, dest_account_number,
		       amount, currency, fee, description,
		       status, response_code, response_msg,
		       created_at, updated_at, completed_at
		FROM transactions
		WHERE transaction_id = $1
	`

	var txn models.Transaction
	var completedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, transactionID).Scan(
		&txn.TransactionID,
		&txn.ReferenceID,
		&txn.IdempotencyKey,
		&txn.SourceBankCode,
		&txn.SourceAccountNumber,
		&txn.DestBankCode,
		&txn.DestAccountNumber,
		&txn.Amount,
		&txn.Currency,
		&txn.Fee,
		&txn.Description,
		&txn.Status,
		&txn.ResponseCode,
		&txn.ResponseMsg,
		&txn.CreatedAt,
		&txn.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		r.logger.ErrorContext("Failed to fetch transaction", err, map[string]interface{}{
			"transactionId": transactionID,
		})
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	if completedAt.Valid {
		txn.CompletedAt = &completedAt.Time
	}

	return &txn, nil
}

// FindByReferenceID retrieves a transaction by reference ID
func (r *transactionRepository) FindByReferenceID(ctx context.Context, referenceID string) (*models.Transaction, error) {
	query := `
		SELECT transaction_id, reference_id, idempotency_key,
		       source_bank_code, source_account_number,
		       dest_bank_code, dest_account_number,
		       amount, currency, fee, description,
		       status, response_code, response_msg,
		       created_at, updated_at, completed_at
		FROM transactions
		WHERE reference_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var txn models.Transaction
	var completedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, referenceID).Scan(
		&txn.TransactionID,
		&txn.ReferenceID,
		&txn.IdempotencyKey,
		&txn.SourceBankCode,
		&txn.SourceAccountNumber,
		&txn.DestBankCode,
		&txn.DestAccountNumber,
		&txn.Amount,
		&txn.Currency,
		&txn.Fee,
		&txn.Description,
		&txn.Status,
		&txn.ResponseCode,
		&txn.ResponseMsg,
		&txn.CreatedAt,
		&txn.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		r.logger.ErrorContext("Failed to fetch transaction", err, map[string]interface{}{
			"referenceId": referenceID,
		})
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	if completedAt.Valid {
		txn.CompletedAt = &completedAt.Time
	}

	return &txn, nil
}

// FindByIdempotencyKey retrieves a transaction by idempotency key
func (r *transactionRepository) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.Transaction, error) {
	query := `
		SELECT transaction_id, reference_id, idempotency_key,
		       source_bank_code, source_account_number,
		       dest_bank_code, dest_account_number,
		       amount, currency, fee, description,
		       status, response_code, response_msg,
		       created_at, updated_at, completed_at
		FROM transactions
		WHERE idempotency_key = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var txn models.Transaction
	var completedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, idempotencyKey).Scan(
		&txn.TransactionID,
		&txn.ReferenceID,
		&txn.IdempotencyKey,
		&txn.SourceBankCode,
		&txn.SourceAccountNumber,
		&txn.DestBankCode,
		&txn.DestAccountNumber,
		&txn.Amount,
		&txn.Currency,
		&txn.Fee,
		&txn.Description,
		&txn.Status,
		&txn.ResponseCode,
		&txn.ResponseMsg,
		&txn.CreatedAt,
		&txn.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		r.logger.ErrorContext("Failed to fetch transaction", err, map[string]interface{}{
			"idempotencyKey": idempotencyKey,
		})
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	if completedAt.Valid {
		txn.CompletedAt = &completedAt.Time
	}

	return &txn, nil
}

// FindAll retrieves all transactions with pagination
func (r *transactionRepository) FindAll(ctx context.Context, limit, offset int) ([]*models.Transaction, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM transactions`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		r.logger.Error("Failed to count transactions", err)
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Get transactions
	query := `
		SELECT transaction_id, reference_id, idempotency_key,
		       source_bank_code, source_account_number,
		       dest_bank_code, dest_account_number,
		       amount, currency, fee, description,
		       status, response_code, response_msg,
		       created_at, updated_at, completed_at
		FROM transactions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to fetch transactions", err)
		return nil, 0, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]*models.Transaction, 0)
	for rows.Next() {
		var txn models.Transaction
		var completedAt sql.NullTime

		err := rows.Scan(
			&txn.TransactionID,
			&txn.ReferenceID,
			&txn.IdempotencyKey,
			&txn.SourceBankCode,
			&txn.SourceAccountNumber,
			&txn.DestBankCode,
			&txn.DestAccountNumber,
			&txn.Amount,
			&txn.Currency,
			&txn.Fee,
			&txn.Description,
			&txn.Status,
			&txn.ResponseCode,
			&txn.ResponseMsg,
			&txn.CreatedAt,
			&txn.UpdatedAt,
			&completedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan transaction", err)
			continue
		}

		if completedAt.Valid {
			txn.CompletedAt = &completedAt.Time
		}

		transactions = append(transactions, &txn)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating transactions", err)
		return nil, 0, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, total, nil
}

// GetStatistics retrieves transaction statistics
func (r *transactionRepository) GetStatistics(ctx context.Context) (*models.TransactionStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_transactions,
			COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END) as completed_count,
			COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed_count,
			COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending_count,
			COUNT(CASE WHEN status = 'PROCESSING' THEN 1 END) as processing_count,
			COALESCE(SUM(CAST(amount AS DECIMAL)), 0) as total_amount,
			COALESCE(SUM(CAST(fee AS DECIMAL)), 0) as total_fee
		FROM transactions
	`

	var stats models.TransactionStatistics
	var totalAmount, totalFee float64

	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalTransactions,
		&stats.CompletedCount,
		&stats.FailedCount,
		&stats.PendingCount,
		&stats.ProcessingCount, // ✅ NEW: Scan processing count separately
		&totalAmount,
		&totalFee,
	)

	if err != nil {
		r.logger.Error("Failed to fetch statistics", err)
		return nil, fmt.Errorf("failed to fetch statistics: %w", err)
	}

	// Format amounts
	stats.TotalAmount = fmt.Sprintf("%.2f", totalAmount)
	stats.TotalFee = fmt.Sprintf("%.2f", totalFee)

	// ✅ NEW: Set SuccessCount as alias of CompletedCount
	stats.SuccessCount = stats.CompletedCount

	return &stats, nil
}

// Delete deletes a transaction by ID
func (r *transactionRepository) Delete(ctx context.Context, transactionID string) error {
	query := `DELETE FROM transactions WHERE transaction_id = $1`

	result, err := r.db.Exec(ctx, query, transactionID)
	if err != nil {
		r.logger.ErrorContext("Failed to delete transaction", err, map[string]interface{}{
			"transactionId": transactionID,
		})
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	r.logger.InfoContext("Transaction deleted successfully", map[string]interface{}{
		"transactionId": transactionID,
	})

	return nil
}

// DeleteAll deletes all transactions
func (r *transactionRepository) DeleteAll(ctx context.Context) error {
	query := `DELETE FROM transactions`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("Failed to delete all transactions", err)
		return fmt.Errorf("failed to delete all transactions: %w", err)
	}

	rowsAffected := result.RowsAffected()
	r.logger.WarnContext("All transactions deleted", map[string]interface{}{
		"rowsDeleted": rowsAffected,
	})

	return nil
}
