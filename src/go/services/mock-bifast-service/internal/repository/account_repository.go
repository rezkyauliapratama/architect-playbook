// src/go/services/mock-bifast-service/internal/repository/account_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"
)

// AccountRepository defines account data access methods
type AccountRepository interface {
	GetAccount(ctx context.Context, bankCode, accountNumber string) (*models.Account, error)
	CreateAccount(ctx context.Context, account *models.Account) error
	UpdateBalance(ctx context.Context, bankCode, accountNumber string, newBalance float64) error
	ListAccounts(ctx context.Context, bankCode string, limit, offset int) ([]*models.Account, int, error)
	DeleteAccount(ctx context.Context, bankCode, accountNumber string) error
}

type accountRepository struct {
	db     *pgxpool.Pool
	logger *logger.Logger
}

// NewAccountRepository creates a new account repository instance
func NewAccountRepository(db *pgxpool.Pool, log *logger.Logger) AccountRepository {
	return &accountRepository{
		db:     db,
		logger: log,
	}
}

// GetAccount retrieves an account by bank code and account number
func (r *accountRepository) GetAccount(ctx context.Context, bankCode, accountNumber string) (*models.Account, error) {
	query := `
		SELECT 
			bank_code,
			account_number,
			account_name,
			account_type,
			balance,
			currency,
			status,
			created_at,
			updated_at
		FROM accounts
		WHERE bank_code = $1 AND account_number = $2 AND status = 'active'
	`

	var account models.Account

	err := r.db.QueryRow(ctx, query, bankCode, accountNumber).Scan(
		&account.BankCode,
		&account.AccountNumber,
		&account.AccountName,
		&account.AccountType,
		&account.Balance,
		&account.Currency,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.WarnContext("Account not found", map[string]interface{}{
				"bankCode":      bankCode,
				"accountNumber": accountNumber,
			})
			return nil, fmt.Errorf("account not found")
		}

		r.logger.ErrorContext("Failed to fetch account", err, map[string]interface{}{
			"bankCode":      bankCode,
			"accountNumber": accountNumber,
		})
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	r.logger.InfoContext("Account retrieved successfully", map[string]interface{}{
		"bankCode":      bankCode,
		"accountNumber": accountNumber,
		"accountName":   account.AccountName,
	})

	return &account, nil
}

// CreateAccount creates a new account
func (r *accountRepository) CreateAccount(ctx context.Context, account *models.Account) error {
	query := `
		INSERT INTO accounts (
			bank_code,
			account_number,
			account_name,
			account_type,
			balance,
			currency,
			status,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err := r.db.Exec(ctx, query,
		account.BankCode,
		account.AccountNumber,
		account.AccountName,
		account.AccountType,
		account.Balance,
		account.Currency,
		account.Status,
		account.CreatedAt,
		account.UpdatedAt,
	)

	if err != nil {
		r.logger.ErrorContext("Failed to create account", err, map[string]interface{}{
			"bankCode":      account.BankCode,
			"accountNumber": account.AccountNumber,
			"accountName":   account.AccountName,
		})
		return fmt.Errorf("failed to create account: %w", err)
	}

	r.logger.InfoContext("Account created successfully", map[string]interface{}{
		"bankCode":      account.BankCode,
		"accountNumber": account.AccountNumber,
		"accountName":   account.AccountName,
		"balance":       account.Balance,
	})

	return nil
}

// UpdateBalance updates account balance
func (r *accountRepository) UpdateBalance(ctx context.Context, bankCode, accountNumber string, newBalance float64) error {
	query := `
		UPDATE accounts
		SET balance = $1,
		    updated_at = NOW()
		WHERE bank_code = $2 AND account_number = $3
	`

	result, err := r.db.Exec(ctx, query, newBalance, bankCode, accountNumber)
	if err != nil {
		r.logger.ErrorContext("Failed to update account balance", err, map[string]interface{}{
			"bankCode":      bankCode,
			"accountNumber": accountNumber,
			"newBalance":    newBalance,
		})
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.WarnContext("No account found to update", map[string]interface{}{
			"bankCode":      bankCode,
			"accountNumber": accountNumber,
		})
		return fmt.Errorf("account not found")
	}

	r.logger.InfoContext("Account balance updated", map[string]interface{}{
		"bankCode":      bankCode,
		"accountNumber": accountNumber,
		"newBalance":    newBalance,
	})

	return nil
}

// ListAccounts retrieves all accounts with optional bank code filter and pagination
func (r *accountRepository) ListAccounts(ctx context.Context, bankCode string, limit, offset int) ([]*models.Account, int, error) {
	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if bankCode != "" {
		countQuery = `SELECT COUNT(*) FROM accounts WHERE bank_code = $1`
		countArgs = []interface{}{bankCode}

		query = `
			SELECT 
				bank_code,
				account_number,
				account_name,
				account_type,
				balance,
				currency,
				status,
				created_at,
				updated_at
			FROM accounts
			WHERE bank_code = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		queryArgs = []interface{}{bankCode, limit, offset}
	} else {
		countQuery = `SELECT COUNT(*) FROM accounts`
		countArgs = []interface{}{}

		query = `
			SELECT 
				bank_code,
				account_number,
				account_name,
				account_type,
				balance,
				currency,
				status,
				created_at,
				updated_at
			FROM accounts
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		queryArgs = []interface{}{limit, offset}
	}

	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		r.logger.ErrorContext("Failed to count accounts", err, map[string]interface{}{
			"bankCode": bankCode,
		})
		return nil, 0, fmt.Errorf("failed to count accounts: %w", err)
	}

	// Get accounts
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		r.logger.ErrorContext("Failed to list accounts", err, map[string]interface{}{
			"bankCode": bankCode,
			"limit":    limit,
			"offset":   offset,
		})
		return nil, 0, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*models.Account, 0)
	for rows.Next() {
		var account models.Account
		err := rows.Scan(
			&account.BankCode,
			&account.AccountNumber,
			&account.AccountName,
			&account.AccountType,
			&account.Balance,
			&account.Currency,
			&account.Status,
			&account.CreatedAt,
			&account.UpdatedAt,
		)

		if err != nil {
			r.logger.ErrorContext("Failed to scan account row", err, map[string]interface{}{
				"bankCode": bankCode,
			})
			continue
		}

		accounts = append(accounts, &account)
	}

	if err := rows.Err(); err != nil {
		r.logger.ErrorContext("Error iterating account rows", err, map[string]interface{}{
			"bankCode": bankCode,
		})
		return nil, 0, fmt.Errorf("error iterating account rows: %w", err)
	}

	r.logger.InfoContext("Accounts listed successfully", map[string]interface{}{
		"bankCode":  bankCode,
		"total":     total,
		"retrieved": len(accounts),
		"limit":     limit,
		"offset":    offset,
	})

	return accounts, total, nil
}

// DeleteAccount deletes an account (soft delete by setting status to INACTIVE)
func (r *accountRepository) DeleteAccount(ctx context.Context, bankCode, accountNumber string) error {
	query := `
		UPDATE accounts
		SET status = 'inactive',
		    updated_at = NOW()
		WHERE bank_code = $1 AND account_number = $2
	`

	result, err := r.db.Exec(ctx, query, bankCode, accountNumber)
	if err != nil {
		r.logger.ErrorContext("Failed to delete account", err, map[string]interface{}{
			"bankCode":      bankCode,
			"accountNumber": accountNumber,
		})
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.WarnContext("No account found to delete", map[string]interface{}{
			"bankCode":      bankCode,
			"accountNumber": accountNumber,
		})
		return fmt.Errorf("account not found")
	}

	r.logger.InfoContext("Account deleted successfully", map[string]interface{}{
		"bankCode":      bankCode,
		"accountNumber": accountNumber,
	})

	return nil
}
