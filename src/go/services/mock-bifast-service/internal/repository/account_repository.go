package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"
)

// AccountRepository defines account data access methods
type AccountRepository interface {
	// GetAccount retrieves account by bank code and account number
	GetAccount(ctx context.Context, bankCode, accountNumber string) (*models.Account, error)

	// CreateAccount creates a new account
	CreateAccount(ctx context.Context, account *models.Account) error

	// UpdateAccount updates an existing account
	UpdateAccount(ctx context.Context, account *models.Account) error

	// DeleteAccount deletes an account
	DeleteAccount(ctx context.Context, bankCode, accountNumber string) error

	// ListAccounts retrieves all accounts with pagination
	ListAccounts(ctx context.Context, limit, offset int) ([]*models.Account, int, error)
}

type accountRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *pgxpool.Pool, logger zerolog.Logger) AccountRepository {
	return &accountRepository{
		db:     db,
		logger: logger,
	}
}

// GetAccount retrieves account by bank code and account number
func (r *accountRepository) GetAccount(ctx context.Context, bankCode, accountNumber string) (*models.Account, error) {
	query := `
		SELECT bank_code, account_number, account_name, account_type,
		       balance, currency, status, created_at, updated_at
		FROM accounts
		WHERE bank_code = $1 AND account_number = $2
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
			return nil, fmt.Errorf("account not found")
		}
		r.logger.Error().Err(err).
			Str("bankCode", bankCode).
			Str("accountNumber", accountNumber).
			Msg("Failed to fetch account")
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	return &account, nil
}

// CreateAccount creates a new account
func (r *accountRepository) CreateAccount(ctx context.Context, account *models.Account) error {
	query := `
		INSERT INTO accounts (
			bank_code, account_number, account_name, account_type,
			balance, currency, status, created_at, updated_at
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
		r.logger.Error().Err(err).
			Str("bankCode", account.BankCode).
			Str("accountNumber", account.AccountNumber).
			Msg("Failed to create account")
		return fmt.Errorf("failed to create account: %w", err)
	}

	r.logger.Info().
		Str("bankCode", account.BankCode).
		Str("accountNumber", account.AccountNumber).
		Msg("Account created successfully")
	return nil
}

// UpdateAccount updates an existing account
func (r *accountRepository) UpdateAccount(ctx context.Context, account *models.Account) error {
	query := `
		UPDATE accounts
		SET account_name = $1,
		    account_type = $2,
		    balance = $3,
		    currency = $4,
		    status = $5,
		    updated_at = $6
		WHERE bank_code = $7 AND account_number = $8
	`

	result, err := r.db.Exec(ctx, query,
		account.AccountName,
		account.AccountType,
		account.Balance,
		account.Currency,
		account.Status,
		account.UpdatedAt,
		account.BankCode,
		account.AccountNumber,
	)

	if err != nil {
		r.logger.Error().Err(err).
			Str("bankCode", account.BankCode).
			Str("accountNumber", account.AccountNumber).
			Msg("Failed to update account")
		return fmt.Errorf("failed to update account: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("account not found")
	}

	r.logger.Info().
		Str("bankCode", account.BankCode).
		Str("accountNumber", account.AccountNumber).
		Msg("Account updated successfully")
	return nil
}

// DeleteAccount deletes an account
func (r *accountRepository) DeleteAccount(ctx context.Context, bankCode, accountNumber string) error {
	query := `DELETE FROM accounts WHERE bank_code = $1 AND account_number = $2`

	result, err := r.db.Exec(ctx, query, bankCode, accountNumber)
	if err != nil {
		r.logger.Error().Err(err).
			Str("bankCode", bankCode).
			Str("accountNumber", accountNumber).
			Msg("Failed to delete account")
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("account not found")
	}

	r.logger.Info().
		Str("bankCode", bankCode).
		Str("accountNumber", accountNumber).
		Msg("Account deleted successfully")
	return nil
}

// ListAccounts retrieves all accounts with pagination
func (r *accountRepository) ListAccounts(ctx context.Context, limit, offset int) ([]*models.Account, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM accounts`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		r.logger.Error().Err(err).Msg("Failed to count accounts")
		return nil, 0, fmt.Errorf("failed to count accounts: %w", err)
	}

	// Get accounts
	query := `
		SELECT bank_code, account_number, account_name, account_type,
		       balance, currency, status, created_at, updated_at
		FROM accounts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to fetch accounts")
		return nil, 0, fmt.Errorf("failed to fetch accounts: %w", err)
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
			r.logger.Error().Err(err).Msg("Failed to scan account")
			continue
		}

		accounts = append(accounts, &account)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating accounts")
		return nil, 0, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, total, nil
}
