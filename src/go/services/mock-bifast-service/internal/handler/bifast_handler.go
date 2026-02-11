package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

// BiFastHandler handles HTTP requests for BI-FAST operations
type BiFastHandler struct {
	service service.BiFastService
	logger  zerolog.Logger
}

// NewBiFastHandler creates a new BiFastHandler
func NewBiFastHandler(service service.BiFastService, logger zerolog.Logger) *BiFastHandler {
	return &BiFastHandler{
		service: service,
		logger:  logger,
	}
}

// AccountInquiry handles account inquiry requests
func (h *BiFastHandler) AccountInquiry(c *fiber.Ctx) error {
	var req dto.AccountInquiryRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse account inquiry request")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Invalid request format",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Validate request
	validationResult := dto.ValidateAccountInquiryRequest(&req)
	if !validationResult.Valid {
		h.logger.Warn().Interface("errors", validationResult.Errors).Msg("Account inquiry validation failed")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ValidationErrorResponse{
			Success: false,
			Error:   "Validation failed",
			Details: validationResult.Errors,
		})
	}

	// Sanitize inputs
	req.BankCode = dto.SanitizeBankCode(req.BankCode)
	req.AccountNumber = dto.SanitizeAccountNumber(req.AccountNumber)

	// Call service
	resp, err := h.service.AccountInquiry(c.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Account inquiry failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			Error:        "Internal server error",
			Message:      err.Error(),
			ResponseCode: dto.ResponseCodeSystemError,
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Return response
	statusCode := fiber.StatusOK
	if resp.ResponseCode != dto.ResponseCodeSuccess {
		statusCode = fiber.StatusBadRequest
	}

	return c.Status(statusCode).JSON(resp)
}

// BiFastTransfer handles BI-FAST transfer requests
func (h *BiFastHandler) BiFastTransfer(c *fiber.Ctx) error {
	var req dto.TransferRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse transfer request")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Invalid request format",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Get idempotency key from header
	idempotencyKey := c.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		h.logger.Warn().Msg("Missing idempotency key")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			Error:        "Missing idempotency key",
			Message:      "X-Idempotency-Key header is required",
			ResponseCode: dto.ResponseCodeInvalidRequest,
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}
	req.IdempotencyKey = idempotencyKey

	// Validate request
	validationResult := dto.ValidateTransferRequest(&req)
	if !validationResult.Valid {
		h.logger.Warn().Interface("errors", validationResult.Errors).Msg("Transfer validation failed")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ValidationErrorResponse{
			Success: false,
			Error:   "Validation failed",
			Details: validationResult.Errors,
		})
	}

	// Sanitize inputs
	req.SourceBankCode = dto.SanitizeBankCode(req.SourceBankCode)
	req.SourceAccountNumber = dto.SanitizeAccountNumber(req.SourceAccountNumber)
	req.DestBankCode = dto.SanitizeBankCode(req.DestBankCode)
	req.DestAccountNumber = dto.SanitizeAccountNumber(req.DestAccountNumber)
	req.Amount = dto.FormatAmount(req.Amount)

	// Call service
	resp, err := h.service.BiFastTransfer(c.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Transfer failed")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			Error:        "Internal server error",
			Message:      err.Error(),
			ResponseCode: dto.ResponseCodeSystemError,
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Return response
	statusCode := fiber.StatusOK
	if resp.ResponseCode != dto.ResponseCodeSuccess {
		statusCode = fiber.StatusBadRequest
	}

	return c.Status(statusCode).JSON(resp)
}

// TransactionStatus handles transaction status query
func (h *BiFastHandler) TransactionStatus(c *fiber.Ctx) error {
	transactionID := c.Params("transactionId")

	if transactionID == "" {
		h.logger.Warn().Msg("Missing transaction ID")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Missing transaction ID",
			Message:   "Transaction ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Call service
	resp, err := h.service.GetTransactionStatus(c.Context(), transactionID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get transaction status")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Internal server error",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Return response
	statusCode := fiber.StatusOK
	if resp.ResponseCode != dto.ResponseCodeSuccess {
		statusCode = fiber.StatusNotFound
	}

	return c.Status(statusCode).JSON(resp)
}

// ListTransactions handles transaction list query (admin)
func (h *BiFastHandler) ListTransactions(c *fiber.Ctx) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Validate pagination
	page, limit, err := dto.ValidatePagination(page, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Invalid pagination parameters")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Invalid pagination",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Call service
	resp, err := h.service.ListTransactions(c.Context(), page, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list transactions")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Internal server error",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// GetStatistics handles statistics query (admin)
func (h *BiFastHandler) GetStatistics(c *fiber.Ctx) error {
	// Call service
	resp, err := h.service.GetStatistics(c.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get statistics")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Internal server error",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// DeleteTransaction handles transaction deletion (admin)
func (h *BiFastHandler) DeleteTransaction(c *fiber.Ctx) error {
	transactionID := c.Params("transactionId")

	if transactionID == "" {
		h.logger.Warn().Msg("Missing transaction ID")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Missing transaction ID",
			Message:   "Transaction ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Call service
	if err := h.service.DeleteTransaction(c.Context(), transactionID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete transaction")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Internal server error",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.DeleteResponse{
		Success: true,
		Message: "Transaction deleted successfully",
	})
}

// ResetAll handles reset all transactions (admin)
func (h *BiFastHandler) ResetAll(c *fiber.Ctx) error {
	var req dto.ResetAllRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		// If no body, assume direct confirmation
		req.Confirm = true
		req.ConfirmPhrase = "DELETE ALL DATA"
	}

	// Validate confirmation
	if !req.Confirm || req.ConfirmPhrase != "DELETE ALL DATA" {
		h.logger.Warn().Msg("Reset all confirmation failed")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Confirmation required",
			Message:   "You must confirm with 'DELETE ALL DATA' phrase",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Call service
	if err := h.service.ResetAll(c.Context()); err != nil {
		h.logger.Error().Err(err).Msg("Failed to reset all data")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:   false,
			Error:     "Internal server error",
			Message:   err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.DeleteResponse{
		Success: true,
		Message: "All transactions deleted successfully",
	})
}
