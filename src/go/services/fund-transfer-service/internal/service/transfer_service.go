// internal/service/transfer_service.go
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/domain"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/repository"

	"github.com/google/uuid"
)

type TransferService struct {
	transferRepo      repository.TransferRepository
	accountClient     client.AccountClient
	transactionClient client.TransactionClient
	ledgerClient      client.LedgerClient

	// Worker pool for concurrent processing
	workerPool chan struct{}
	maxWorkers int
}

func NewTransferService(
	repo repository.TransferRepository,
	accountClient client.AccountClient,
	transactionClient client.TransactionClient,
	ledgerClient client.LedgerClient,
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
		workerPool:        make(chan struct{}, maxWorkers),
		maxWorkers:        maxWorkers,
	}
}

func (s *TransferService) CreateTransfer(ctx context.Context, req *dto.CreateTransferRequest) (*dto.TransferResponse, error) {
	// Validate source account
	account, err := s.accountClient.GetAccount(ctx, req.SourceAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate source account: %w", err)
	}

	// Check if account has sufficient balance
	if account.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Generate transfer ID and reference number
	transferID := fmt.Sprintf("TRF-%s", uuid.New().String()[:8])
	refNumber := fmt.Sprintf("%s%s", account.BankCode, time.Now().Format("20060102150405"))

	// Calculate fee (could be based on transfer type, amount, etc.)
	var fee float64 = 0
	if req.DestinationBankCode != account.BankCode {
		fee = 6500 // Standard fee for inter-bank transfers
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
	}

	if err := s.transferRepo.Create(ctx, transfer); err != nil {
		return nil, fmt.Errorf("failed to create transfer: %w", err)
	}

	// Process transfer using worker pool for controlled concurrency
	select {
	case s.workerPool <- struct{}{}: // Acquire a worker slot
		go func() {
			defer func() { <-s.workerPool }() // Release the slot when done
			s.processTransfer(context.Background(), transfer)
		}()
	default:
		// Worker pool is full, process synchronously
		go s.processTransfer(context.Background(), transfer)
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

func (s *TransferService) processTransfer(ctx context.Context, transfer *domain.Transfer) {
	// Update status to PROCESSING
	if err := s.transferRepo.UpdateStatus(ctx, transfer.TransferID, domain.TransferStatusProcessing); err != nil {
		// Log error and return
		return
	}

	// Using WaitGroup and mutex for coordinating concurrent operations
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
			mutex.Lock()
			overallSuccess = false
			mutex.Unlock()
			return
		}
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
			mutex.Lock()
			overallSuccess = false
			mutex.Unlock()
			return
		}

		mutex.Lock()
		transactionID = txnResp.TransactionID
		mutex.Unlock()
	}()

	// Concurrent operation 3: Create ledger entries
	wg.Add(1)
	go func() {
		defer wg.Done()

		ledgerReq := &dto.CreateLedgerEntriesRequest{
			Entries: []dto.LedgerEntry{
				{
					AccountingCode: "1100",
					EntryType:      "DEBIT",
					Amount:         transfer.Amount,
					Currency:       transfer.Currency,
					ReferenceType:  "TRANSFER",
					ReferenceID:    transfer.TransferID,
					Description:    "Pendebitan transfer nasabah",
				},
				{
					AccountingCode: "2100",
					EntryType:      "CREDIT",
					Amount:         transfer.Amount,
					Currency:       transfer.Currency,
					ReferenceType:  "TRANSFER",
					ReferenceID:    transfer.TransferID,
					Description:    "Pengkreditan transfer nasabah",
				},
			},
		}

		// Add fee entries if applicable
		if transfer.Fee > 0 {
			ledgerReq.Entries = append(ledgerReq.Entries,
				dto.LedgerEntry{
					AccountingCode: "4300",
					EntryType:      "DEBIT",
					Amount:         transfer.Fee,
					Currency:       transfer.Currency,
					ReferenceType:  "FEE",
					ReferenceID:    transfer.TransferID,
					Description:    "Biaya transfer",
				},
				dto.LedgerEntry{
					AccountingCode: "8100",
					EntryType:      "CREDIT",
					Amount:         transfer.Fee,
					Currency:       transfer.Currency,
					ReferenceType:  "FEE_INCOME",
					ReferenceID:    transfer.TransferID,
					Description:    "Pendapatan biaya transfer",
				},
			)
		}

		ledgerResp, err := s.ledgerClient.CreateEntries(ctx, ledgerReq)
		if err != nil {
			mutex.Lock()
			overallSuccess = false
			mutex.Unlock()
			return
		}

		mutex.Lock()
		journalID = ledgerResp.JournalID
		mutex.Unlock()
	}()

	// Wait for all concurrent operations to complete
	wg.Wait()

	if overallSuccess {
		// Update transfer with transaction IDs and ledger journal ID
		if transactionID != "" {
			if err := s.transferRepo.UpdateTransactionIDs(ctx, transfer.TransferID, []string{transactionID}); err != nil {
				// Log error
				return
			}
		}

		if journalID != "" {
			if err := s.transferRepo.UpdateLedgerJournalID(ctx, transfer.TransferID, journalID); err != nil {
				// Log error
				return
			}
		}

		// Mark transfer as completed
		if err := s.transferRepo.CompleteTransfer(ctx, transfer.TransferID); err != nil {
			// Log error
			return
		}
	} else {
		// Mark transfer as failed if any operation failed
		s.transferRepo.UpdateStatus(ctx, transfer.TransferID, domain.TransferStatusFailed)
		// In a production system, implement compensation transactions here
	}
}

func (s *TransferService) GetTransferByID(ctx context.Context, transferID string) (*dto.TransferResponse, error) {
	transfer, err := s.transferRepo.GetByID(ctx, transferID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}

	if transfer == nil {
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

func (s *TransferService) GetTransfersByAccount(ctx context.Context, accountID string, limit, offset int) ([]*dto.TransferResponse, error) {
	transfers, err := s.transferRepo.GetBySourceAccountID(ctx, accountID, limit, offset)
	if err != nil {
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

	return response, nil
}

func (s *TransferService) CreateBulkTransfer(ctx context.Context, req *dto.CreateBulkTransferRequest) (*dto.BulkTransferResponse, error) {
	// Validate source account
	account, err := s.accountClient.GetAccount(ctx, req.SourceAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate source account: %w", err)
	}

	// Calculate total amount and validate sufficient balance
	var totalAmount float64
	for _, t := range req.Transfers {
		totalAmount += t.Amount
	}

	if account.Balance < totalAmount {
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
	}

	if err := s.transferRepo.CreateBulkTransfer(ctx, bulkTransfer); err != nil {
		return nil, fmt.Errorf("failed to create bulk transfer: %w", err)
	}

	// Process individual transfers concurrently with a semaphore for controlled parallelism
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
			}

			transferResp, err := s.CreateTransfer(ctx, transferReq)
			if err != nil {
				// Log error but continue with other transfers
				return
			}

			// Add to bulk transfer items
			item := &domain.BulkTransferItem{
				BulkTransferID: bulkTransferID,
				TransferID:     transferResp.TransferID,
				Status:         domain.TransferStatusPending,
			}

			if err := s.transferRepo.AddBulkTransferItem(ctx, item); err != nil {
				// Log error
				return
			}

			mutex.Lock()
			transferIDs = append(transferIDs, transferResp.TransferID)
			mutex.Unlock()
		}(item)
	}

	wg.Wait()

	// Update bulk transfer with generated transfer IDs
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
