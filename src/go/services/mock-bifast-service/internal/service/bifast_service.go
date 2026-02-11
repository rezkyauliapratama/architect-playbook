package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
)

// BiFastService defines the business logic for BI-FAST operations
type BiFastService interface {
	// AccountInquiry validates and retrieves account information
	AccountInquiry(ctx context.Context, req *dto.AccountInquiryRequest) (*dto.AccountInquiryResponse, error)

	// BiFastTransfer initiates a BI-FAST transfer
	BiFastTransfer(ctx context.Context, req *dto.TransferRequest) (*dto.TransferResponse, error)

	// GetTransactionStatus retrieves transaction status
	GetTransactionStatus(ctx context.Context, transactionID string) (*dto.TransactionStatusResponse, error)

	// ListTransactions retrieves all transactions with pagination
	ListTransactions(ctx context.Context, page, limit int) (*dto.TransactionListResponse, error)

	// GetStatistics retrieves transaction statistics
	GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error)

	// DeleteTransaction deletes a transaction (admin only)
	DeleteTransaction(ctx context.Context, transactionID string) error

	// ResetAll deletes all transactions (admin only)
	ResetAll(ctx context.Context) error
}

type biFastService struct {
	txnRepo        repository.TransactionRepository
	accRepo        repository.AccountRepository
	notificationCl *client.NotificationClient
	logger         zerolog.Logger
	rand           *rand.Rand
}

// NewBiFastService creates a new instance of BiFastService
func NewBiFastService(
	txnRepo repository.TransactionRepository,
	accRepo repository.AccountRepository,
	notificationCl *client.NotificationClient,
	log zerolog.Logger,
) BiFastService {
	return &biFastService{
		txnRepo:        txnRepo,
		accRepo:        accRepo,
		notificationCl: notificationCl,
		logger:         log,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// AccountInquiry validates and retrieves account information
func (s *biFastService) AccountInquiry(ctx context.Context, req *dto.AccountInquiryRequest) (*dto.AccountInquiryResponse, error) {
	s.logger.Info().
		Str("bankCode", req.BankCode).
		Str("accountNumber", req.AccountNumber).
		Str("referenceId", req.ReferenceID).
		Msg("Processing account inquiry")

	// Simulate processing delay
	time.Sleep(time.Duration(100+s.rand.Intn(200)) * time.Millisecond)

	// Get account from repository
	account, err := s.accRepo.GetAccount(ctx, req.BankCode, req.AccountNumber)
	if err != nil {
		s.logger.Error().Err(err).Msg("Account not found")
		return &dto.AccountInquiryResponse{
			ResponseCode: dto.ResponseCodeAccountNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeAccountNotFound),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Success response
	return &dto.AccountInquiryResponse{
		ResponseCode:  dto.ResponseCodeSuccess,
		ResponseMsg:   dto.GetResponseMessage(dto.ResponseCodeSuccess),
		ReferenceID:   req.ReferenceID,
		BankCode:      req.BankCode,
		AccountNumber: req.AccountNumber,
		AccountName:   account.AccountName,
		AccountType:   account.AccountType,
	}, nil
}

// BiFastTransfer initiates a BI-FAST transfer
func (s *biFastService) BiFastTransfer(ctx context.Context, req *dto.TransferRequest) (*dto.TransferResponse, error) {
	s.logger.Info().
		Str("referenceId", req.ReferenceID).
		Str("amount", req.Amount).
		Str("sourceAccount", req.SourceAccountNumber).
		Str("destAccount", req.DestAccountNumber).
		Msg("Initiating BI-FAST transfer")

	// Validate source account
	sourceAccount, err := s.accRepo.GetAccount(ctx, req.SourceBankCode, req.SourceAccountNumber)
	if err != nil {
		s.logger.Error().Err(err).Msg("Source account not found")
		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeAccountNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeAccountNotFound),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Validate destination account
	destAccount, err := s.accRepo.GetAccount(ctx, req.DestBankCode, req.DestAccountNumber)
	if err != nil {
		s.logger.Error().Err(err).Msg("Destination account not found")
		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeAccountNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeAccountNotFound),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Check if same account
	if req.SourceBankCode == req.DestBankCode && req.SourceAccountNumber == req.DestAccountNumber {
		s.logger.Warn().Msg("Source and destination accounts are the same")
		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeInvalidTransaction,
			ResponseMsg:  "Source and destination accounts cannot be the same",
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("BIFAST-%s", uuid.New().String())

	// Calculate fee (flat fee of IDR 2,500 for this mock)
	fee := "2500.00"

	// Create transaction record
	transaction := &models.Transaction{
		TransactionID:       transactionID,
		ReferenceID:         req.ReferenceID,
		IdempotencyKey:      req.IdempotencyKey,
		SourceBankCode:      req.SourceBankCode,
		SourceAccountNumber: req.SourceAccountNumber,
		DestBankCode:        req.DestBankCode,
		DestAccountNumber:   req.DestAccountNumber,
		Amount:              req.Amount,
		Currency:            req.Currency,
		Fee:                 fee,
		Description:         req.Description,
		Status:              string(models.StatusPending),
		ResponseCode:        dto.ResponseCodeSuccess,
		ResponseMsg:         "Transaction initiated",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Save transaction
	if err := s.txnRepo.Create(ctx, transaction); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create transaction")
		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeSystemError),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Simulate async processing with random delay (1-2 seconds)
	processingTime := time.Duration(1000+s.rand.Intn(1000)) * time.Millisecond

	// Async processing simulation
	go func() {
		time.Sleep(processingTime)

		// Update to PROCESSING
		s.txnRepo.UpdateStatus(
			context.Background(),
			transactionID,
			models.StatusProcessing,
			dto.ResponseCodeSuccess,
			"Processing transfer",
			nil,
		)

		// Complete after 500ms
		time.Sleep(500 * time.Millisecond)
		completedAt := time.Now()
		s.txnRepo.UpdateStatus(
			context.Background(),
			transactionID,
			models.StatusCompleted,
			dto.ResponseCodeSuccess,
			dto.GetResponseMessage(dto.ResponseCodeSuccess),
			&completedAt,
		)

		// Fetch updated transaction for notification
		updatedTxn, err := s.txnRepo.FindByID(context.Background(), transactionID)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("transactionId", transactionID).
				Msg("Failed to fetch transaction for notification")
			return
		}

		// Send notification (non-blocking)
		go func() {
			notifCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.notificationCl.SendTransactionNotification(notifCtx, updatedTxn); err != nil {
				s.logger.Warn().
					Err(err).
					Str("transactionId", transactionID).
					Msg("Failed to send transaction notification")
			}
		}()

		s.logger.Info().
			Str("transactionId", transactionID).
			Str("amount", req.Amount).
			Msg("Transfer completed successfully")
	}()

	// Return immediate response
	return &dto.TransferResponse{
		ResponseCode:      dto.ResponseCodeSuccess,
		ResponseMsg:       dto.GetResponseMessage(dto.ResponseCodeSuccess),
		TransactionID:     transactionID,
		ReferenceID:       req.ReferenceID,
		Amount:            req.Amount,
		Currency:          req.Currency,
		Fee:               fee,
		SourceAccountName: sourceAccount.AccountName,
		DestAccountName:   destAccount.AccountName,
		Status:            string(models.StatusPending),
		TransactionTime:   time.Now().Format(time.RFC3339),
	}, nil
}

// GetTransactionStatus retrieves transaction status
func (s *biFastService) GetTransactionStatus(ctx context.Context, transactionID string) (*dto.TransactionStatusResponse, error) {
	s.logger.Debug().
		Str("transactionId", transactionID).
		Msg("Fetching transaction status")

	// Fetch transaction from repository
	txn, err := s.txnRepo.FindByID(ctx, transactionID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Transaction not found")
		return &dto.TransactionStatusResponse{
			ResponseCode: dto.ResponseCodeTransactionNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeTransactionNotFound),
		}, nil
	}

	// Build response
	response := &dto.TransactionStatusResponse{
		ResponseCode:        dto.ResponseCodeSuccess,
		ResponseMsg:         dto.GetResponseMessage(dto.ResponseCodeSuccess),
		TransactionID:       txn.TransactionID,
		ReferenceID:         txn.ReferenceID,
		SourceBankCode:      txn.SourceBankCode,
		SourceAccountNumber: txn.SourceAccountNumber,
		DestBankCode:        txn.DestBankCode,
		DestAccountNumber:   txn.DestAccountNumber,
		Amount:              txn.Amount,
		Currency:            txn.Currency,
		Fee:                 txn.Fee,
		Description:         txn.Description,
		Status:              txn.Status,
		TransactionTime:     txn.CreatedAt.Format(time.RFC3339),
	}

	if txn.CompletedAt != nil {
		completedTime := txn.CompletedAt.Format(time.RFC3339)
		response.CompletedTime = &completedTime
	}

	return response, nil
}

// ListTransactions retrieves all transactions with pagination
func (s *biFastService) ListTransactions(ctx context.Context, page, limit int) (*dto.TransactionListResponse, error) {
	s.logger.Debug().
		Int("page", page).
		Int("limit", limit).
		Msg("Listing transactions")

	// Calculate offset
	offset := (page - 1) * limit

	// Fetch transactions
	transactions, total, err := s.txnRepo.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch transactions")
		return nil, err
	}

	// Build response
	items := make([]dto.TransactionItem, len(transactions))
	for i, txn := range transactions {
		item := dto.TransactionItem{
			TransactionID:       txn.TransactionID,
			ReferenceID:         txn.ReferenceID,
			SourceBankCode:      txn.SourceBankCode,
			SourceAccountNumber: txn.SourceAccountNumber,
			DestBankCode:        txn.DestBankCode,
			DestAccountNumber:   txn.DestAccountNumber,
			Amount:              txn.Amount,
			Currency:            txn.Currency,
			Fee:                 txn.Fee,
			Status:              txn.Status,
			CreatedAt:           txn.CreatedAt.Format(time.RFC3339),
		}

		if txn.CompletedAt != nil {
			completedTime := txn.CompletedAt.Format(time.RFC3339)
			item.CompletedAt = &completedTime
		}

		items[i] = item
	}

	return &dto.TransactionListResponse{
		Success: true,
		Data:    items,
		Pagination: dto.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	}, nil
}

// GetStatistics retrieves transaction statistics
func (s *biFastService) GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error) {
	s.logger.Debug().Msg("Fetching statistics")

	stats, err := s.txnRepo.GetStatistics(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch statistics")
		return nil, err
	}

	return &dto.StatisticsResponse{
		Success: true,
		Data: dto.Statistics{
			TotalTransactions: stats.TotalTransactions,
			CompletedCount:    stats.CompletedCount,
			FailedCount:       stats.FailedCount,
			PendingCount:      stats.PendingCount,
			TotalAmount:       stats.TotalAmount,
			TotalFee:          stats.TotalFee,
		},
	}, nil
}

// DeleteTransaction deletes a transaction (admin only)
func (s *biFastService) DeleteTransaction(ctx context.Context, transactionID string) error {
	s.logger.Info().
		Str("transactionId", transactionID).
		Msg("Deleting transaction")

	return s.txnRepo.Delete(ctx, transactionID)
}

// ResetAll deletes all transactions (admin only)
func (s *biFastService) ResetAll(ctx context.Context) error {
	s.logger.Warn().Msg("Resetting all transactions")
	return s.txnRepo.DeleteAll(ctx)
}
