// internal/repository/postgres/bifast_repository.go
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/domain"
)

type bifastRepository struct {
	db *sqlx.DB
}

func NewBifastRepository(db *sqlx.DB) *bifastRepository {
	return &bifastRepository{db: db}
}

// Bank Account methods
func (r *bifastRepository) GetAccountByNumber(ctx context.Context, accountNumber string) (*domain.BankAccount, error) {
	log := logger.Get().WithField("method", "GetAccountByNumber").WithField("accountNumber", accountNumber)

	query := `SELECT * FROM bank_accounts WHERE account_number = $1`

	var account domain.BankAccount
	err := r.db.GetContext(ctx, &account, query, accountNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("Account not found")
			return nil, nil
		}
		log.Error("Failed to get account", err)
		return nil, err
	}

	return &account, nil
}

func (r *bifastRepository) GetAccountByProxy(ctx context.Context, proxyType domain.ProxyType, proxyValue string) (*domain.BankAccount, error) {
	log := logger.Get().WithField("method", "GetAccountByProxy").
		WithField("proxyType", proxyType).
		WithField("proxyValue", proxyValue)

	query := `SELECT * FROM bank_accounts WHERE proxy_type = $1 AND proxy_value = $2`

	var account domain.BankAccount
	err := r.db.GetContext(ctx, &account, query, proxyType, proxyValue)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("Account not found")
			return nil, nil
		}
		log.Error("Failed to get account by proxy", err)
		return nil, err
	}

	return &account, nil
}

func (r *bifastRepository) CreateAccount(ctx context.Context, account *domain.BankAccount) error {
	log := logger.Get().WithField("method", "CreateAccount").WithField("accountNumber", account.AccountNumber)

	query := `
        INSERT INTO bank_accounts (
            id, account_number, account_name, bank_code, bank_name, 
            proxy_type, proxy_value, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8
        )
    `

	_, err := r.db.ExecContext(
		ctx, query,
		account.ID, account.AccountNumber, account.AccountName,
		account.BankCode, account.BankName, account.ProxyType,
		account.ProxyValue, account.CreatedAt,
	)

	if err != nil {
		log.Error("Failed to create account", err)
		return err
	}

	log.Info("Account created successfully")
	return nil
}

// Transaction methods
func (r *bifastRepository) CreateTransaction(ctx context.Context, transaction *domain.BifastTransaction) error {
	log := logger.Get().WithField("method", "CreateTransaction").WithField("transactionId", transaction.TransactionID)

	query := `
        INSERT INTO bifast_transactions (
            id, transaction_id, source_account_number, source_account_name, source_bank_code,
            destination_account_number, destination_account_name, destination_bank_code,
            amount, fee, currency, status, reference_id, description, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
        )
    `

	_, err := r.db.ExecContext(
		ctx, query,
		transaction.ID, transaction.TransactionID, transaction.SourceAccountNumber,
		transaction.SourceAccountName, transaction.SourceBankCode, transaction.DestinationAccountNumber,
		transaction.DestinationAccountName, transaction.DestinationBankCode, transaction.Amount,
		transaction.Fee, transaction.Currency, transaction.Status, transaction.ReferenceID,
		transaction.Description, transaction.CreatedAt, transaction.UpdatedAt,
	)

	if err != nil {
		log.Error("Failed to create transaction", err)
		return err
	}

	log.Info("Transaction created successfully")
	return nil
}

func (r *bifastRepository) GetTransactionByID(ctx context.Context, transactionID string) (*domain.BifastTransaction, error) {
	log := logger.Get().WithField("method", "GetTransactionByID").WithField("transactionId", transactionID)

	query := `SELECT * FROM bifast_transactions WHERE transaction_id = $1`

	var transaction domain.BifastTransaction
	err := r.db.GetContext(ctx, &transaction, query, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("Transaction not found")
			return nil, nil
		}
		log.Error("Failed to get transaction", err)
		return nil, err
	}

	return &transaction, nil
}

func (r *bifastRepository) UpdateTransactionStatus(ctx context.Context, transactionID string, status domain.TransactionStatus) error {
	log := logger.Get().WithField("method", "UpdateTransactionStatus").
		WithField("transactionId", transactionID).
		WithField("status", status)

	query := `
        UPDATE bifast_transactions 
        SET status = $1, updated_at = $2
        WHERE transaction_id = $3
    `

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, status, now, transactionID)
	if err != nil {
		log.Error("Failed to update transaction status", err)
		return err
	}

	log.Info("Transaction status updated successfully")
	return nil
}

func (r *bifastRepository) CompleteTransaction(ctx context.Context, transactionID string) error {
	log := logger.Get().WithField("method", "CompleteTransaction").WithField("transactionId", transactionID)

	query := `
        UPDATE bifast_transactions 
        SET status = $1, updated_at = $2, completed_at = $2
        WHERE transaction_id = $3
    `

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, domain.TransactionStatusCompleted, now, transactionID)
	if err != nil {
		log.Error("Failed to complete transaction", err)
		return err
	}

	log.Info("Transaction completed successfully")
	return nil
}
