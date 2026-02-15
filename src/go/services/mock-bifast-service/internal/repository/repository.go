// internal/repository/repository.go
package repository

import (
	"context"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/domain"
)

// BifastRepository defines the interface for BI-Fast repository operations
type BifastRepository interface {
	// Bank Account methods
	GetAccountByNumber(ctx context.Context, accountNumber string) (*domain.BankAccount, error)
	GetAccountByProxy(ctx context.Context, proxyType domain.ProxyType, proxyValue string) (*domain.BankAccount, error)
	CreateAccount(ctx context.Context, account *domain.BankAccount) error

	// Transaction methods
	CreateTransaction(ctx context.Context, transaction *domain.BifastTransaction) error
	GetTransactionByID(ctx context.Context, transactionID string) (*domain.BifastTransaction, error)
	UpdateTransactionStatus(ctx context.Context, transactionID string, status domain.TransactionStatus) error
	CompleteTransaction(ctx context.Context, transactionID string) error
}
