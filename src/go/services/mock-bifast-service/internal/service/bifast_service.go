// internal/service/bifast_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/uuid"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/domain"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
)

type BifastService struct {
	repo               repository.BifastRepository
	notificationClient client.NotificationClient
	defaultFee         float64
	maxAmount          float64
	minAmount          float64
	successRate        int
}

func NewBifastService(repo repository.BifastRepository, notificationClient client.NotificationClient, defaultFee, maxAmount, minAmount float64, successRate int) *BifastService {
	return &BifastService{
		repo:               repo,
		notificationClient: notificationClient,
		defaultFee:         defaultFee,
		maxAmount:          maxAmount,
		minAmount:          minAmount,
		successRate:        successRate,
	}
}

// AccountInquiry handles inquiries for account validation
func (s *BifastService) AccountInquiry(ctx context.Context, req *dto.AccountInquiryRequest) (*dto.AccountInquiryResponse, error) {
	log := logger.Get().WithField("method", "BifastService.AccountInquiry")

	var account *domain.BankAccount
	var err error

	// Check if we're looking up by account number or proxy
	if req.AccountNumber != "" {
		account, err = s.repo.GetAccountByNumber(ctx, req.AccountNumber)
	} else if req.ProxyType != "" && req.ProxyValue != "" {
		account, err = s.repo.GetAccountByProxy(ctx, domain.ProxyType(req.ProxyType), req.ProxyValue)
	} else {
		return nil, errors.New("either account number or proxy information is required")
	}

	if err != nil {
		log.Error("Failed to get account", err)
		return nil, fmt.Errorf("inquiry failed: %w", err)
	}

	if account == nil {
		log.Warn("Account not found")
		return nil, errors.New("account not found")
	}

	// If bank code filter is provided, check if it matches
	if req.BankCode != "" && account.BankCode != req.BankCode {
		log.Warn("Account bank code doesn't match requested bank code")
		return nil, errors.New("account not found in specified bank")
	}

	log.Info(fmt.Sprint("Account inquiry successful", map[string]interface{}{
		"accountNumber": account.AccountNumber,
		"accountName":   account.AccountName,
	}))

	// Return the account information
	return &dto.AccountInquiryResponse{
		AccountNumber: account.AccountNumber,
		AccountName:   account.AccountName,
		BankCode:      account.BankCode,
		BankName:      account.BankName,
		ProxyType:     account.GetProxyType(),
		ProxyValue:    account.GetProxyValue(),
	}, nil
}

// BifastTransfer handles the BI-Fast transfer
func (s *BifastService) BifastTransfer(ctx context.Context, req *dto.BifastTransferRequest) (*dto.BifastTransferResponse, error) {
	log := logger.Get().WithField("method", "BifastService.BifastTransfer")

	// Validate the transfer amount (BI-Fast has limits)
	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if req.Amount < s.minAmount {
		return nil, fmt.Errorf("amount below BI-FAST minimum of Rp %.2f", s.minAmount)
	}

	if req.Amount > s.maxAmount {
		return nil, fmt.Errorf("amount exceeds BI-FAST maximum of Rp %.2f", s.maxAmount)
	}

	// Validate destination account exists (in real BI-FAST, this would be an online inquiry)
	destAccount, err := s.repo.GetAccountByNumber(ctx, req.DestinationAccountNumber)
	if err != nil {
		log.Error("Failed to validate destination account", err)
		return nil, fmt.Errorf("destination account validation failed: %w", err)
	}

	if destAccount == nil {
		log.Warn("Destination account not found")
		return nil, errors.New("destination account not found")
	}

	// Ensure bank code matches
	if destAccount.BankCode != req.DestinationBankCode {
		log.Warn("Destination bank code doesn't match account bank code")
		return nil, errors.New("destination bank code does not match account")
	}

	// Generate a unique transaction ID with time ordering benefits
	transactionID := fmt.Sprintf("BIFAST-%s", uuid.Generate()[:8])
	now := time.Now()

	// Get destination account name from our records if not provided or validate it
	destinationName := destAccount.AccountName
	if req.DestinationAccountName != "" && req.DestinationAccountName != destinationName {
		log.Info("Destination name in request doesn't match, using validated name")
	}

	// Create a transaction record
	transaction := &domain.BifastTransaction{
		ID:                       uuid.Generate(),
		TransactionID:            transactionID,
		SourceAccountNumber:      req.SourceAccountNumber,
		SourceAccountName:        req.SourceAccountName,
		SourceBankCode:           req.SourceBankCode,
		DestinationAccountNumber: req.DestinationAccountNumber,
		DestinationAccountName:   destinationName,
		DestinationBankCode:      destAccount.BankCode,
		Amount:                   req.Amount,
		Fee:                      s.defaultFee,
		Currency:                 "IDR", // BI-Fast only supports IDR
		Status:                   domain.TransactionStatusPending,
		ReferenceID:              req.ReferenceID,
		Description:              req.Description,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	// Save the transaction
	err = s.repo.CreateTransaction(ctx, transaction)
	if err != nil {
		log.Error("Failed to create transaction", err)
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	// Simulate processing (asynchronously complete the transaction)
	go s.processTransaction(transactionID)

	log.Info(fmt.Sprint("BI-Fast transfer initiated", map[string]interface{}{
		"transactionId":      transactionID,
		"amount":             req.Amount,
		"sourceAccount":      req.SourceAccountNumber,
		"destinationAccount": req.DestinationAccountNumber,
	}))

	// Return the initial response
	return &dto.BifastTransferResponse{
		TransactionID:      transactionID,
		Amount:             req.Amount,
		Fee:                s.defaultFee,
		Status:             string(domain.TransactionStatusPending),
		ReferenceID:        req.ReferenceID,
		SourceAccount:      req.SourceAccountNumber,
		DestinationAccount: req.DestinationAccountNumber,
		TransactionTime:    now,
	}, nil
}

// TransactionStatus gets the status of a transaction
func (s *BifastService) TransactionStatus(ctx context.Context, req *dto.TransactionStatusRequest) (*dto.TransactionStatusResponse, error) {
	log := logger.Get().WithField("method", "BifastService.TransactionStatus").
		WithField("transactionId", req.TransactionID)

	// Get the transaction
	transaction, err := s.repo.GetTransactionByID(ctx, req.TransactionID)
	if err != nil {
		log.Error("Failed to get transaction", err)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if transaction == nil {
		log.Warn("Transaction not found")
		return nil, errors.New("transaction not found")
	}

	log.Info(fmt.Sprint("Transaction status retrieved", map[string]interface{}{
		"status": transaction.Status,
	}))

	// Return the status
	return &dto.TransactionStatusResponse{
		TransactionID:      transaction.TransactionID,
		Status:             string(transaction.Status),
		Amount:             transaction.Amount,
		Fee:                transaction.Fee,
		SourceAccount:      transaction.SourceAccountNumber,
		DestinationAccount: transaction.DestinationAccountNumber,
		ReferenceID:        transaction.ReferenceID,
		TransactionTime:    transaction.CreatedAt,
		CompletedTime:      transaction.CompletedAt,
	}, nil
}

// Helper method to process a transaction asynchronously
func (s *BifastService) processTransaction(transactionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log := logger.Get().WithField("method", "BifastService.processTransaction").
		WithField("transactionId", transactionID)

	// Simulate processing delay (BI-Fast usually completes in under 3 seconds)
	processingTime := time.Duration(rand.Intn(2000)+500) * time.Millisecond
	time.Sleep(processingTime)

	// Get transaction details for notification
	transaction, err := s.repo.GetTransactionByID(ctx, transactionID)
	if err != nil {
		log.Error("Failed to get transaction for processing", err)
		return
	}

	if transaction == nil {
		log.Info("Transaction not found for processing")
		return
	}

	// Simulate success rate (configurable, typically high for BI-Fast)
	isSuccess := rand.Intn(100) < s.successRate

	var newStatus domain.TransactionStatus
	if isSuccess {
		// Complete the transaction
		err = s.repo.CompleteTransaction(ctx, transactionID)
		newStatus = domain.TransactionStatusCompleted
	} else {
		// Fail the transaction
		err = s.repo.UpdateTransactionStatus(ctx, transactionID, domain.TransactionStatusFailed)
		newStatus = domain.TransactionStatusFailed
	}

	if err != nil {
		log.Error("Failed to update transaction", err)
		return
	}

	// Log the result
	if isSuccess {
		log.Info(fmt.Sprint("Transaction completed successfully", map[string]interface{}{
			"processingTime": processingTime.Milliseconds(),
		}))
	} else {
		log.Warn(fmt.Sprint("Transaction failed", map[string]interface{}{
			"processingTime": processingTime.Milliseconds(),
		}))
	}

	// Send notification (don't block on this)
	statusResponse := &dto.TransactionStatusResponse{
		TransactionID:      transaction.TransactionID,
		Status:             string(newStatus),
		Amount:             transaction.Amount,
		Fee:                transaction.Fee,
		SourceAccount:      transaction.SourceAccountNumber,
		DestinationAccount: transaction.DestinationAccountNumber,
		ReferenceID:        transaction.ReferenceID,
		TransactionTime:    transaction.CreatedAt,
	}

	if isSuccess {
		completedAt := time.Now()
		statusResponse.CompletedTime = &completedAt
	}

	go func() {
		notificationCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := s.notificationClient.SendTransferNotification(notificationCtx, statusResponse)
		if err != nil {
			log.Error("Failed to send transfer notification", err)
		}
	}()
}
