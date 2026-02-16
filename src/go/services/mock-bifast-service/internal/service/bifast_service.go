// src/go/services/mock-bifast-service/internal/service/bifast_service.go
package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/uuid"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
)

// BiFastService defines the business logic interface for BI-FAST operations
type BiFastService interface {
	AccountInquiry(ctx context.Context, req *dto.AccountInquiryRequest) (*dto.AccountInquiryResponse, error)
	BiFastTransfer(ctx context.Context, req *dto.TransferRequest) (*dto.TransferResponse, error)
	GetTransactionStatus(ctx context.Context, transactionID string) (*dto.TransactionStatusResponse, error)
	ListTransactions(ctx context.Context, page, limit int) (*dto.TransactionListResponse, error)
	GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error)
	DeleteTransaction(ctx context.Context, transactionID string) error
	ResetAll(ctx context.Context) error
}

type biFastService struct {
	txnRepo        repository.TransactionRepository
	accRepo        repository.AccountRepository
	notificationCl *client.NotificationClient
	config         config.BiFastConfig
	logger         *logger.Logger
	rand           *rand.Rand
}

// NewBiFastService creates a new instance of BiFastService
func NewBiFastService(
	txnRepo repository.TransactionRepository,
	accRepo repository.AccountRepository,
	notificationCl *client.NotificationClient,
	bifastConfig config.BiFastConfig,
	log *logger.Logger,
) BiFastService {
	return &biFastService{
		txnRepo:        txnRepo,
		accRepo:        accRepo,
		notificationCl: notificationCl,
		config:         bifastConfig,
		logger:         log,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// AccountInquiry validates and retrieves account information
func (s *biFastService) AccountInquiry(ctx context.Context, req *dto.AccountInquiryRequest) (*dto.AccountInquiryResponse, error) {
	// ✅ Use InfoContext with structured logging
	s.logger.InfoContext("Processing account inquiry", map[string]interface{}{
		"bankCode":      req.BankCode,
		"accountNumber": req.AccountNumber,
		"referenceId":   req.ReferenceID,
	})

	// Simulate processing delay (100-300ms)
	time.Sleep(time.Duration(100+s.rand.Intn(200)) * time.Millisecond)

	// Get account from repository
	account, err := s.accRepo.GetAccount(ctx, req.BankCode, req.AccountNumber)
	if err != nil {
		// ✅ Use ErrorContext with proper error logging
		s.logger.ErrorContext("Account not found", err, map[string]interface{}{
			"bankCode":      req.BankCode,
			"accountNumber": req.AccountNumber,
		})

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
	// ✅ Use InfoContext with all relevant fields
	s.logger.InfoContext("Initiating BI-FAST transfer", map[string]interface{}{
		"referenceId":    req.ReferenceID,
		"amount":         req.Amount,
		"sourceAccount":  req.SourceAccountNumber,
		"destAccount":    req.DestAccountNumber,
		"sourceBankCode": req.SourceBankCode,
		"destBankCode":   req.DestBankCode,
	})

	// Validate amount format
	amount, err := dto.ParseAmount(req.Amount)
	if err != nil {
		s.logger.ErrorContext("Invalid amount format", err, map[string]interface{}{
			"amount":      req.Amount,
			"referenceId": req.ReferenceID,
		})

		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeInvalidAmount,
			ResponseMsg:  "Invalid amount format",
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Check minimum amount
	if amount < s.config.MinAmount {
		s.logger.WarnContext("Amount below minimum limit", map[string]interface{}{
			"amount":      amount,
			"minAmount":   s.config.MinAmount,
			"referenceId": req.ReferenceID,
		})

		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeInvalidAmount,
			ResponseMsg:  fmt.Sprintf("Amount below BI-FAST minimum of Rp %.2f", s.config.MinAmount),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Check maximum amount
	if amount > s.config.MaxAmount {
		s.logger.WarnContext("Amount exceeds maximum limit", map[string]interface{}{
			"amount":      amount,
			"maxAmount":   s.config.MaxAmount,
			"referenceId": req.ReferenceID,
		})

		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeInvalidAmount,
			ResponseMsg:  fmt.Sprintf("Amount exceeds BI-FAST maximum of Rp %.2f", s.config.MaxAmount),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Validate source account
	sourceAccount, err := s.accRepo.GetAccount(ctx, req.SourceBankCode, req.SourceAccountNumber)
	if err != nil {
		s.logger.ErrorContext("Source account not found", err, map[string]interface{}{
			"bankCode":      req.SourceBankCode,
			"accountNumber": req.SourceAccountNumber,
		})

		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeAccountNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeAccountNotFound),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Validate destination account
	destAccount, err := s.accRepo.GetAccount(ctx, req.DestBankCode, req.DestAccountNumber)
	if err != nil {
		s.logger.ErrorContext("Destination account not found", err, map[string]interface{}{
			"bankCode":      req.DestBankCode,
			"accountNumber": req.DestAccountNumber,
		})

		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeAccountNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeAccountNotFound),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Check if same account
	if req.SourceBankCode == req.DestBankCode && req.SourceAccountNumber == req.DestAccountNumber {
		s.logger.Warn("Source and destination accounts are the same")
		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeInvalidTransaction,
			ResponseMsg:  "Source and destination accounts cannot be the same",
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// ✅ Generate transaction ID using libs/uuid (UUID v7 - time-ordered)
	transactionID := fmt.Sprintf("BIFAST-%s", uuid.Generate())

	// Use configured fee
	fee := fmt.Sprintf("%.2f", s.config.Fee)

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
		s.logger.ErrorContext("Failed to create transaction", err, map[string]interface{}{
			"transactionId": transactionID,
			"referenceId":   req.ReferenceID,
		})

		return &dto.TransferResponse{
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeSystemError),
			ReferenceID:  req.ReferenceID,
		}, nil
	}

	// Simulate async processing
	processingTime := time.Duration(1000+s.rand.Intn(1000)) * time.Millisecond

	// Async processing simulation
	go func() {
		// ✅ Use WithTransferID for tracing throughout async operation
		logWithTransfer := s.logger.WithTransferID(transactionID)

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

		logWithTransfer.InfoContext("Transaction processing started", map[string]interface{}{
			"amount":      req.Amount,
			"referenceId": req.ReferenceID,
		})

		// Complete after 500ms with success rate logic
		time.Sleep(500 * time.Millisecond)

		// Apply success rate for testing
		isSuccess := s.rand.Intn(100) < s.config.SuccessRate

		completedAt := time.Now()
		var finalStatus models.TransactionStatus
		var responseCode string
		var responseMsg string

		if isSuccess {
			finalStatus = models.StatusCompleted
			responseCode = dto.ResponseCodeSuccess
			responseMsg = dto.GetResponseMessage(dto.ResponseCodeSuccess)
		} else {
			finalStatus = models.StatusFailed
			responseCode = dto.ResponseCodeSystemError
			responseMsg = "Transaction failed (simulated failure for testing)"
		}

		s.txnRepo.UpdateStatus(
			context.Background(),
			transactionID,
			finalStatus,
			responseCode,
			responseMsg,
			&completedAt,
		)

		// Fetch updated transaction for notification
		updatedTxn, err := s.txnRepo.FindByID(context.Background(), transactionID)
		if err != nil {
			logWithTransfer.ErrorContext("Failed to fetch transaction for notification", err, map[string]interface{}{
				"referenceId": req.ReferenceID,
			})
			return
		}

		// Send notification (non-blocking)
		go func() {
			notifCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.notificationCl.SendTransactionNotification(notifCtx, updatedTxn); err != nil {
				logWithTransfer.WarnContext("Failed to send transaction notification", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}()

		logWithTransfer.InfoContext("Transfer processing completed", map[string]interface{}{
			"amount":  req.Amount,
			"success": isSuccess,
			"status":  string(finalStatus),
		})
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

// GetTransactionStatus retrieves transaction status by ID
func (s *biFastService) GetTransactionStatus(ctx context.Context, transactionID string) (*dto.TransactionStatusResponse, error) {
	// ✅ Use WithTransferID for consistent tracing
	logWithTransfer := s.logger.WithTransferID(transactionID)
	logWithTransfer.Info("Fetching transaction status")

	// Fetch transaction from repository
	txn, err := s.txnRepo.FindByID(ctx, transactionID)
	if err != nil {
		logWithTransfer.Error("Transaction not found", err)
		return &dto.TransactionStatusResponse{
			ResponseCode: dto.ResponseCodeTransactionNotFound,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeTransactionNotFound),
		}, nil
	}

	// Build response
	response := &dto.TransactionStatusResponse{
		ResponseCode:  dto.ResponseCodeSuccess,
		ResponseMsg:   dto.GetResponseMessage(dto.ResponseCodeSuccess),
		TransactionID: txn.TransactionID,
		ReferenceID:   txn.ReferenceID,
		Amount:        txn.Amount,
		Currency:      txn.Currency,
		Fee:           txn.Fee,
		Description:   txn.Description,
		Status:        txn.Status,
		CreatedAt:     txn.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     txn.UpdatedAt.Format(time.RFC3339),
	}

	// Add completion time if available
	if txn.CompletedAt != nil {
		completedAtStr := txn.CompletedAt.Format(time.RFC3339)
		response.CompletedAt = &completedAtStr
	}

	return response, nil
}

// ListTransactions retrieves paginated list of transactions
func (s *biFastService) ListTransactions(ctx context.Context, page, limit int) (*dto.TransactionListResponse, error) {
	s.logger.InfoContext("Fetching transaction list", map[string]interface{}{
		"page":  page,
		"limit": limit,
	})

	// ✅ Calculate offset from page and limit
	offset := (page - 1) * limit

	// Validate pagination parameters
	validatedPage, validatedLimit, err := dto.ValidatePagination(page, limit)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination parameters: %w", err)
	}

	// Recalculate offset with validated values
	offset = (validatedPage - 1) * validatedLimit

	// Use FindAll method
	transactions, total, err := s.txnRepo.FindAll(ctx, validatedLimit, offset)
	if err != nil {
		s.logger.Error("Failed to list transactions", err)
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	transactionItems := make([]*dto.TransactionItem, len(transactions))
	for i, txn := range transactions {
		transactionItems[i] = dto.ConvertFromModel(txn)
	}

	// Calculate total pages (ceiling division)
	totalPages := (total + validatedLimit - 1) / validatedLimit
	if totalPages == 0 {
		totalPages = 1
	}

	// Build response
	return &dto.TransactionListResponse{
		Success: true,
		Data:    transactionItems,
		Pagination: dto.PaginationMeta{
			Page:       validatedPage,
			Limit:      validatedLimit,
			TotalItems: total,
			TotalPages: totalPages,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// GetStatistics retrieves transaction statistics
func (s *biFastService) GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error) {
	s.logger.Info("Fetching transaction statistics")

	// Get statistics from repository
	dbStats, err := s.txnRepo.GetStatistics(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch statistics", err)
		return nil, fmt.Errorf("failed to fetch statistics: %w", err)
	}

	// Calculate success rate
	var successRate float64
	if dbStats.TotalTransactions > 0 {
		successRate = (float64(dbStats.SuccessCount) / float64(dbStats.TotalTransactions)) * 100
	}

	// ✅ Use correct field names from updated model
	stats := &dto.StatisticsResponse{
		Success:           true,
		TotalTransactions: dbStats.TotalTransactions,
		SuccessCount:      dbStats.SuccessCount,
		FailedCount:       dbStats.FailedCount,
		PendingCount:      dbStats.PendingCount,
		ProcessingCount:   dbStats.ProcessingCount,
		TotalAmount:       dbStats.TotalAmount,
		TotalFee:          dbStats.TotalFee,
		SuccessRate:       successRate,
		Timestamp:         time.Now().Format(time.RFC3339),
	}

	return stats, nil
}

// DeleteTransaction deletes a transaction by ID (admin only)
func (s *biFastService) DeleteTransaction(ctx context.Context, transactionID string) error {
	logWithTransfer := s.logger.WithTransferID(transactionID)
	logWithTransfer.Warn("Deleting transaction (admin operation)")

	// Delete from repository
	if err := s.txnRepo.Delete(ctx, transactionID); err != nil {
		logWithTransfer.Error("Failed to delete transaction", err)
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	logWithTransfer.Info("Transaction deleted successfully")
	return nil
}

// ResetAll deletes all transactions (admin only - testing)
func (s *biFastService) ResetAll(ctx context.Context) error {
	s.logger.Warn("Resetting all transactions (admin operation)")

	// Delete all transactions
	if err := s.txnRepo.DeleteAll(ctx); err != nil {
		s.logger.Error("Failed to reset all transactions", err)
		return fmt.Errorf("failed to reset all transactions: %w", err)
	}

	s.logger.Info("All transactions reset successfully")
	return nil
}
