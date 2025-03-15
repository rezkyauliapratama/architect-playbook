// internal/service/transfer_service.go
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/domain"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/repository"
)

// TransferService handles fund transfer business logic
type TransferService struct {
	transferRepo      repository.TransferRepository
	accountClient     client.AccountClient
	transactionClient client.TransactionClient
	ledgerClient      client.LedgerClient
	config            *config.Config

	// Worker pool for concurrent processing
	workerPool chan struct{}
	maxWorkers int
}

// NewTransferService creates a new transfer service
func NewTransferService(
	repo repository.TransferRepository,
	accountClient client.AccountClient,
	transactionClient client.TransactionClient,
	ledgerClient client.LedgerClient,
	config *config.Config,
	maxWorkers int,
) *TransferService {
	// Default to 10 workers if not specified
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	return &TransferService{
		transferRepo:      repo,
		accountClient:     accountClient,
		transactionClient: transactionClient,
		ledgerClient:      ledgerClient,
		config:            config,
		workerPool:        make(chan struct{}, maxWorkers),
		maxWorkers:        maxWorkers,
	}
}

// CreateTransfer creates a new fund transfer
func (s *TransferService) CreateTransfer(ctx context.Context, req *dto.CreateTransferRequest) (*dto.TransferResponse, error) {
	log := logger.Get().WithFields(map[string]interface{}{
		"sourceAccountId":          req.SourceAccountID,
		"destinationAccountNumber": req.DestinationAccountNumber,
		"amount":                   req.Amount,
		"currency":                 req.Currency,
	})

	log.Info("Creating new transfer")

	// Validate source account
	account, err := s.accountClient.GetAccount(ctx, req.SourceAccountID)
	if err != nil {
		log.Error("Failed to validate source account", err)
		return nil, fmt.Errorf("failed to validate source account: %w", err)
	}

	// Check if account has sufficient balance
	if account.Balance < req.Amount {
		log.Warn("Insufficient balance for transfer")
		return nil, fmt.Errorf("insufficient balance")
	}

	// Generate transfer ID and reference number
	transferID := fmt.Sprintf("TRF-%s", uuid.New().String()[:8])
	refNumber := fmt.Sprintf("%s%s", account.BankCode, time.Now().Format("20060102150405"))

	// Calculate fee based on request or default logic
	var fee float64 = 0
	if req.FeeType == "FLAT" {
		if req.CustomFee != nil {
			fee = *req.CustomFee
		} else {
			fee = s.config.DefaultFlatFee // Default from configuration
		}
	} else if req.FeeType == "PERCENTAGE" {
		if req.CustomFee != nil {
			fee = req.Amount * (*req.CustomFee) / 100
		} else {
			fee = req.Amount * s.config.DefaultPercentageFee / 100
		}
	} else if req.DestinationBankCode != account.BankCode {
		fee = s.config.DefaultFlatFee // Default inter-bank fee
	}

	// Create transfer record
	transfer := &domain.Transfer{
		TransferID:               transferID,
		SourceAccountID:          req.SourceAccountID,
		DestinationAccountNumber: req.DestinationAccountNumber,
		DestinationBankCode:      req.DestinationBankCode,
		DestinationAccountName:   req.DestinationAccountName,
		Amount:                   req.Amount,
		Currency:                 req.Currency,
		Status:                   domain.TransferStatusPending,
		ReferenceNumber:          refNumber,
		Fee:                      fee,
		Description:              req.Description,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := s.transferRepo.Create(ctx, transfer); err != nil {
		log.Error("Failed to create transfer record", err)
		return nil, fmt.Errorf("failed to create transfer: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"transferId": transferID,
		"fee":        fee,
	}).Info("Transfer created successfully")

	// Process transfer using worker pool for controlled concurrency
	select {
	case s.workerPool <- struct{}{}: // Acquire a worker slot
		go func() {
			defer func() { <-s.workerPool }() // Release the slot when done
			s.processTransfer(context.Background(), transfer, req)
		}()
	default:
		// Worker pool is full, process in a separate goroutine anyway
		// but without the worker pool management
		go s.processTransfer(context.Background(), transfer, req)
	}

	return &dto.TransferResponse{
		TransferID:               transfer.TransferID,
		SourceAccountID:          transfer.SourceAccountID,
		DestinationAccountNumber: transfer.DestinationAccountNumber,
		DestinationBankCode:      transfer.DestinationBankCode,
		DestinationAccountName:   transfer.DestinationAccountName,
		Amount:                   transfer.Amount,
		Currency:                 transfer.Currency,
		Status:                   string(transfer.Status),
		ReferenceNumber:          transfer.ReferenceNumber,
		Fee:                      transfer.Fee,
		Description:              transfer.Description,
		CreatedAt:                transfer.CreatedAt,
	}, nil
}

// processTransfer handles the actual transfer processing asynchronously
func (s *TransferService) processTransfer(ctx context.Context, transfer *domain.Transfer, req *dto.CreateTransferRequest) {
	log := logger.Get().WithFields(map[string]interface{}{
		"transferId":               transfer.TransferID,
		"sourceAccountId":          transfer.SourceAccountID,
		"destinationAccountNumber": transfer.DestinationAccountNumber,
		"amount":                   transfer.Amount,
	})

	log.Info("Processing transfer")

	// Update status to PROCESSING
	if err := s.transferRepo.UpdateStatus(ctx, transfer.TransferID, domain.TransferStatusProcessing); err != nil {
		log.Error("Failed to update transfer status to PROCESSING", err)
		return
	}

	// Using WaitGroup for concurrent operations
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var overallSuccess = true
	var transactionID string
	var journalID string

	// Concurrent operation 1: Update account balance
	wg.Add(1)
	go func() {
		defer wg.Done()

		debitReq := &dto.UpdateAccountBalanceRequest{
			Adjustment:  -1 * (transfer.Amount + transfer.Fee),
			ReferenceID: transfer.TransferID,
			Description: fmt.Sprintf("Transfer ke %s: %s", transfer.DestinationAccountNumber, transfer.Description),
		}

		_, err := s.accountClient.UpdateBalance(ctx, transfer.SourceAccountID, debitReq)
		if err != nil {
			log.Error("Failed to debit source account", err)
			mutex.Lock()
			overallSuccess = false
			mutex.Unlock()
			return
		}

		log.Info("Successfully debited source account")
	}()

	// Concurrent operation 2: Create transaction record
	wg.Add(1)
	go func() {
		defer wg.Done()

		txnReq := &dto.CreateTransactionRequest{
			AccountID:   transfer.SourceAccountID,
			Type:        "DEBIT",
			Amount:      transfer.Amount + transfer.Fee,
			Currency:    transfer.Currency,
			ReferenceID: transfer.TransferID,
			Description: fmt.Sprintf("Transfer ke %s: %s", transfer.DestinationAccountNumber, transfer.Description),
		}

		txnResp, err := s.transactionClient.CreateTransaction(ctx, txnReq)
		if err != nil {
			log.Error("Failed to create transaction record", err)
			mutex.Lock()
			overallSuccess = false
			mutex.Unlock()
			return
		}

		mutex.Lock()
		transactionID = txnResp.TransactionID
		mutex.Unlock()

		log.Info("Successfully created transaction record")
	}()

	// Concurrent operation 3: Create ledger entries - WITH DYNAMIC VALUES
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get accounting codes from request or fall back to defaults
		debitAccountCode := s.config.DefaultDebitAccountCode
		creditAccountCode := s.config.DefaultCreditAccountCode
		feeDebitAccountCode := s.config.DefaultFeeDebitAccountCode
		feeCreditAccountCode := s.config.DefaultFeeCreditAccountCode

		// Override with request values if present
		if req != nil {
			if req.DebitAccountCode != "" {
				debitAccountCode = req.DebitAccountCode
			}
			if req.CreditAccountCode != "" {
				creditAccountCode = req.CreditAccountCode
			}
			if req.FeeDebitAccountCode != "" {
				feeDebitAccountCode = req.FeeDebitAccountCode
			}
			if req.FeeCreditAccountCode != "" {
				feeCreditAccountCode = req.FeeCreditAccountCode
			}
		}

		// Get descriptions from request or fall back to defaults
		debitDescription := s.config.DefaultDebitDescription
		creditDescription := s.config.DefaultCreditDescription
		feeDebitDescription := s.config.DefaultFeeDebitDescription
		feeCreditDescription := s.config.DefaultFeeCreditDescription

		// Override with request values if present
		if req != nil {
			if req.DebitDescription != "" {
				debitDescription = req.DebitDescription
			}
			if req.CreditDescription != "" {
				creditDescription = req.CreditDescription
			}
			if req.FeeDebitDescription != "" {
				feeDebitDescription = req.FeeDebitDescription
			}
			if req.FeeCreditDescription != "" {
				feeCreditDescription = req.FeeCreditDescription
			}
		}

		referenceType := "TRANSFER"
		if req != nil && req.ReferenceType != "" {
			referenceType = req.ReferenceType
		}

		ledgerEntries := []dto.LedgerEntry{
			{
				AccountingCode: debitAccountCode,
				EntryType:      "DEBIT",
				Amount:         transfer.Amount,
				Currency:       transfer.Currency,
				ReferenceType:  referenceType,
				ReferenceID:    transfer.TransferID,
				Description:    debitDescription,
			},
			{
				AccountingCode: creditAccountCode,
				EntryType:      "CREDIT",
				Amount:         transfer.Amount,
				Currency:       transfer.Currency,
				ReferenceType:  referenceType,
				ReferenceID:    transfer.TransferID,
				Description:    creditDescription,
			},
		}

		// Add fee entries if applicable
		if transfer.Fee > 0 {
			ledgerEntries = append(ledgerEntries,
				dto.LedgerEntry{
					AccountingCode: feeDebitAccountCode,
					EntryType:      "DEBIT",
					Amount:         transfer.Fee,
					Currency:       transfer.Currency,
					ReferenceType:  "FEE",
					ReferenceID:    transfer.TransferID,
					Description:    feeDebitDescription,
				},
				dto.LedgerEntry{
					AccountingCode: feeCreditAccountCode,
					EntryType:      "CREDIT",
					Amount:         transfer.Fee,
					Currency:       transfer.Currency,
					ReferenceType:  "FEE_INCOME",
					ReferenceID:    transfer.TransferID,
					Description:    feeCreditDescription,
				},
			)
		}

		ledgerReq := &dto.CreateLedgerEntriesRequest{
			Entries: ledgerEntries,
		}

		ledgerResp, err := s.ledgerClient.CreateEntries(ctx, ledgerReq)
		if err != nil {
			log.Error("Failed to create ledger entries", err)
			mutex.Lock()
			overallSuccess = false
			mutex.Unlock()
			return
		}

		mutex.Lock()
		journalID = ledgerResp.JournalID
		mutex.Unlock()

		log.WithFields(map[string]interface{}{
			"journalId":  ledgerResp.JournalID,
			"entryCount": len(ledgerEntries),
		}).Info("Successfully created ledger entries")
	}()

	// Wait for all concurrent operations to complete
	wg.Wait()

	if overallSuccess {
		// Update transfer with transaction and ledger information
		if transactionID != "" {
			if err := s.transferRepo.UpdateTransactionIDs(ctx, transfer.TransferID, []string{transactionID}); err != nil {
				log.Error("Failed to update transfer with transaction IDs", err)
				return
			}
		}

		if journalID != "" {
			if err := s.transferRepo.UpdateLedgerJournalID(ctx, transfer.TransferID, journalID); err != nil {
				log.Error("Failed to update transfer with ledger journal ID", err)
				return
			}
		}

		// Mark transfer as completed
		if err := s.transferRepo.CompleteTransfer(ctx, transfer.TransferID); err != nil {
			log.Error("Failed to complete transfer", err)
			return
		}

		log.Info("Transfer completed successfully")
	} else {
		// Mark transfer as failed
		s.transferRepo.UpdateStatus(ctx, transfer.TransferID, domain.TransferStatusFailed)
		log.Error("Transfer failed due to one or more errors", nil)

		// TODO: Implement compensation logic here
		// This would involve crediting back the account if it was debited
		// but other operations failed
	}
}

// GetTransferByID retrieves a transfer by ID
func (s *TransferService) GetTransferByID(ctx context.Context, transferID string) (*dto.TransferResponse, error) {
	log := logger.Get().With("transferId", transferID)

	log.Info("Retrieving transfer details")

	transfer, err := s.transferRepo.GetByID(ctx, transferID)
	if err != nil {
		log.Error("Failed to get transfer", err)
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}

	if transfer == nil {
		log.Warn("Transfer not found")
		return nil, fmt.Errorf("transfer not found")
	}

	return &dto.TransferResponse{
		TransferID:               transfer.TransferID,
		SourceAccountID:          transfer.SourceAccountID,
		DestinationAccountNumber: transfer.DestinationAccountNumber,
		DestinationBankCode:      transfer.DestinationBankCode,
		DestinationAccountName:   transfer.DestinationAccountName,
		Amount:                   transfer.Amount,
		Currency:                 transfer.Currency,
		Status:                   string(transfer.Status),
		ReferenceNumber:          transfer.ReferenceNumber,
		Fee:                      transfer.Fee,
		Description:              transfer.Description,
		CreatedAt:                transfer.CreatedAt,
		CompletedAt:              transfer.CompletedAt,
	}, nil
}

// GetTransfersByAccount retrieves transfers for a specific account
func (s *TransferService) GetTransfersByAccount(ctx context.Context, accountID string, limit, offset int) ([]*dto.TransferResponse, error) {
	log := logger.Get().With("accountId", accountID)

	log.Info("Retrieving transfers for account")

	transfers, err := s.transferRepo.GetBySourceAccountID(ctx, accountID, limit, offset)
	if err != nil {
		log.Error("Failed to get transfers for account", err)
		return nil, fmt.Errorf("failed to get transfers: %w", err)
	}

	var response []*dto.TransferResponse
	for _, transfer := range transfers {
		response = append(response, &dto.TransferResponse{
			TransferID:               transfer.TransferID,
			SourceAccountID:          transfer.SourceAccountID,
			DestinationAccountNumber: transfer.DestinationAccountNumber,
			DestinationBankCode:      transfer.DestinationBankCode,
			DestinationAccountName:   transfer.DestinationAccountName,
			Amount:                   transfer.Amount,
			Currency:                 transfer.Currency,
			Status:                   string(transfer.Status),
			ReferenceNumber:          transfer.ReferenceNumber,
			Fee:                      transfer.Fee,
			Description:              transfer.Description,
			CreatedAt:                transfer.CreatedAt,
			CompletedAt:              transfer.CompletedAt,
		})
	}

	log.With("count", len(response)).Info("Successfully retrieved transfers for account")

	return response, nil
}

// CreateBulkTransfer creates multiple transfers in a single batch
func (s *TransferService) CreateBulkTransfer(ctx context.Context, req *dto.CreateBulkTransferRequest) (*dto.BulkTransferResponse, error) {
	log := logger.Get().WithFields(map[string]interface{}{
		"sourceAccountId": req.SourceAccountID,
		"transferCount":   len(req.Transfers),
		"batchReference":  req.BatchReference,
	})

	log.Info("Creating bulk transfer")

	// Validate source account
	account, err := s.accountClient.GetAccount(ctx, req.SourceAccountID)
	if err != nil {
		log.Error("Failed to validate source account for bulk transfer", err)
		return nil, fmt.Errorf("failed to validate source account: %w", err)
	}

	// Calculate total amount
	var totalAmount float64
	for _, transfer := range req.Transfers {
		totalAmount += transfer.Amount
	}

	// Check if account has sufficient balance
	if account.Balance < totalAmount {
		log.Warn("Insufficient balance for bulk transfer")
		return nil, fmt.Errorf("insufficient balance for bulk transfer")
	}

	// Generate bulk transfer ID
	bulkTransferID := fmt.Sprintf("BULK-%s", uuid.New().String()[:8])

	// Create bulk transfer record
	bulkTransfer := &domain.BulkTransfer{
		BulkTransferID:  bulkTransferID,
		SourceAccountID: req.SourceAccountID,
		TotalAmount:     totalAmount,
		TransferCount:   len(req.Transfers),
		Currency:        req.Currency,
		Status:          domain.TransferStatusPending,
		BatchReference:  req.BatchReference,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.transferRepo.CreateBulkTransfer(ctx, bulkTransfer); err != nil {
		log.Error("Failed to create bulk transfer record", err)
		return nil, fmt.Errorf("failed to create bulk transfer: %w", err)
	}

	// Process transfers concurrently with a semaphore
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.maxWorkers)

	var transferIDs []string
	var mutex sync.Mutex

	for _, item := range req.Transfers {
		wg.Add(1)

		// Acquire semaphore slot
		semaphore <- struct{}{}

		go func(transferItem dto.BulkTransferItem) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release slot when done

			// Create individual transfer
			transferReq := &dto.CreateTransferRequest{
				SourceAccountID:          req.SourceAccountID,
				DestinationAccountNumber: transferItem.DestinationAccountNumber,
				DestinationBankCode:      transferItem.DestinationBankCode,
				DestinationAccountName:   transferItem.DestinationAccountName,
				Amount:                   transferItem.Amount,
				Currency:                 req.Currency,
				Description:              transferItem.Description,
				// Pass through any accounting code overrides if present
				DebitAccountCode:     req.DebitAccountCode,
				CreditAccountCode:    req.CreditAccountCode,
				FeeDebitAccountCode:  req.FeeDebitAccountCode,
				FeeCreditAccountCode: req.FeeCreditAccountCode,
				DebitDescription:     req.DebitDescription,
				CreditDescription:    req.CreditDescription,
				FeeDebitDescription:  req.FeeDebitDescription,
				FeeCreditDescription: req.FeeCreditDescription,
				ReferenceType:        req.ReferenceType,
			}

			transferResp, err := s.CreateTransfer(ctx, transferReq)
			if err != nil {
				log.Error("Failed to create individual transfer in bulk operation", err)
				return
			}

			// Add to bulk transfer items
			item := &domain.BulkTransferItem{
				BulkTransferID: bulkTransferID,
				TransferID:     transferResp.TransferID,
				Status:         domain.TransferStatusPending,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			if err := s.transferRepo.AddBulkTransferItem(ctx, item); err != nil {
				log.Error("Failed to add item to bulk transfer", err)
				return
			}

			mutex.Lock()
			transferIDs = append(transferIDs, transferResp.TransferID)
			mutex.Unlock()
		}(item)
	}

	wg.Wait()

	// Update bulk transfer status
	s.transferRepo.UpdateBulkTransferStatus(ctx, bulkTransferID, domain.TransferStatusProcessing)

	log.WithFields(map[string]interface{}{
		"bulkTransferId": bulkTransferID,
		"transferCount":  len(transferIDs),
	}).Info("Bulk transfer created successfully")

	// Return response
	response := &dto.BulkTransferResponse{
		BulkTransferID:  bulkTransferID,
		SourceAccountID: req.SourceAccountID,
		TotalAmount:     totalAmount,
		TransferCount:   len(req.Transfers),
		Currency:        req.Currency,
		Status:          string(domain.TransferStatusProcessing),
		BatchReference:  req.BatchReference,
		TransferIDs:     transferIDs,
		CreatedAt:       bulkTransfer.CreatedAt,
	}

	return response, nil
}
