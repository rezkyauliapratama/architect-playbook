// src/go/services/mock-bifast-service/internal/handler/bifast_handler.go
package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

// BiFastHandler handles HTTP requests for BI-FAST operations
type BiFastHandler struct {
	service service.BiFastService
	logger  *logger.Logger // ✅ Use libs/logger
}

// NewBiFastHandler creates a new BI-FAST handler instance
func NewBiFastHandler(svc service.BiFastService, log *logger.Logger) *BiFastHandler {
	return &BiFastHandler{
		service: svc,
		logger:  log,
	}
}

// AccountInquiry handles account inquiry requests
func (h *BiFastHandler) AccountInquiry(c *fiber.Ctx) error {
	// ✅ Extract request ID from context (set by middleware)
	requestID := c.Locals("requestID").(string)

	// ✅ Create logger with request ID for tracing
	logWithRequest := h.logger.WithRequestID(requestID)

	var req dto.AccountInquiryRequest
	if err := c.BodyParser(&req); err != nil {
		logWithRequest.ErrorContext("Invalid request body", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeInvalidRequest,
			ResponseMsg:  "Invalid request body",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logWithRequest.WarnContext("Validation failed", map[string]interface{}{
			"error":         err.Error(),
			"bankCode":      req.BankCode,
			"accountNumber": req.AccountNumber,
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeInvalidRequest,
			ResponseMsg:  "Validation failed",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// ✅ Log incoming request with structured data
	logWithRequest.InfoContext("Account inquiry request received", map[string]interface{}{
		"bankCode":      req.BankCode,
		"accountNumber": req.AccountNumber,
		"referenceId":   req.ReferenceID,
	})

	// Call service layer
	resp, err := h.service.AccountInquiry(c.Context(), &req)
	if err != nil {
		logWithRequest.ErrorContext("Account inquiry failed", err, map[string]interface{}{
			"bankCode":      req.BankCode,
			"accountNumber": req.AccountNumber,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeSystemError),
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Success response
	resp.Success = resp.ResponseCode == dto.ResponseCodeSuccess
	resp.Timestamp = time.Now().Format(time.RFC3339)

	statusCode := fiber.StatusOK
	if resp.ResponseCode != dto.ResponseCodeSuccess {
		statusCode = fiber.StatusNotFound
	}

	return c.Status(statusCode).JSON(resp)
}

// BiFastTransfer handles BI-FAST transfer requests
func (h *BiFastHandler) BiFastTransfer(c *fiber.Ctx) error {
	requestID := c.Locals("requestID").(string)
	logWithRequest := h.logger.WithRequestID(requestID)

	var req dto.TransferRequest
	if err := c.BodyParser(&req); err != nil {
		logWithRequest.ErrorContext("Invalid request body", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeInvalidRequest,
			ResponseMsg:  "Invalid request body",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logWithRequest.WarnContext("Transfer validation failed", map[string]interface{}{
			"error":         err.Error(),
			"referenceId":   req.ReferenceID,
			"amount":        req.Amount,
			"sourceAccount": req.SourceAccountNumber,
			"destAccount":   req.DestAccountNumber,
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeInvalidRequest,
			ResponseMsg:  "Validation failed",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// ✅ Log incoming transfer request
	logWithRequest.InfoContext("BI-FAST transfer request received", map[string]interface{}{
		"referenceId":    req.ReferenceID,
		"idempotencyKey": req.IdempotencyKey,
		"amount":         req.Amount,
		"sourceAccount":  req.SourceAccountNumber,
		"destAccount":    req.DestAccountNumber,
		"sourceBankCode": req.SourceBankCode,
		"destBankCode":   req.DestBankCode,
	})

	// Call service layer
	resp, err := h.service.BiFastTransfer(c.Context(), &req)
	if err != nil {
		logWithRequest.ErrorContext("Transfer processing failed", err, map[string]interface{}{
			"referenceId": req.ReferenceID,
			"amount":      req.Amount,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeSystemError),
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Success response
	resp.Success = resp.ResponseCode == dto.ResponseCodeSuccess
	resp.Timestamp = time.Now().Format(time.RFC3339)

	statusCode := fiber.StatusOK
	if resp.ResponseCode != dto.ResponseCodeSuccess {
		statusCode = fiber.StatusBadRequest
	}

	// ✅ Log successful transfer initiation
	logWithRequest.InfoContext("Transfer initiated successfully", map[string]interface{}{
		"transactionId": resp.TransactionID,
		"referenceId":   resp.ReferenceID,
		"amount":        resp.Amount,
		"status":        resp.Status,
	})

	return c.Status(statusCode).JSON(resp)
}

// TransactionStatus handles transaction status inquiry
func (h *BiFastHandler) TransactionStatus(c *fiber.Ctx) error {
	requestID := c.Locals("requestID").(string)
	logWithRequest := h.logger.WithRequestID(requestID)

	transactionID := c.Params("transactionId")
	if transactionID == "" {
		logWithRequest.Warn("Transaction ID is required")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeInvalidRequest,
			ResponseMsg:  "Transaction ID is required",
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// ✅ Use WithTransferID for transaction tracing
	logWithTransfer := logWithRequest.WithTransferID(transactionID)
	logWithTransfer.Info("Transaction status inquiry")

	// Call service layer
	resp, err := h.service.GetTransactionStatus(c.Context(), transactionID)
	if err != nil {
		logWithTransfer.ErrorContext("Failed to fetch transaction status", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  dto.GetResponseMessage(dto.ResponseCodeSystemError),
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	// Success response
	resp.Success = resp.ResponseCode == dto.ResponseCodeSuccess
	resp.Timestamp = time.Now().Format(time.RFC3339)

	statusCode := fiber.StatusOK
	if resp.ResponseCode == dto.ResponseCodeTransactionNotFound {
		statusCode = fiber.StatusNotFound
	}

	return c.Status(statusCode).JSON(resp)
}

// ListTransactions handles listing transactions (admin endpoint)
func (h *BiFastHandler) ListTransactions(c *fiber.Ctx) error {
	requestID := c.Locals("requestID").(string)
	logWithRequest := h.logger.WithRequestID(requestID)

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// ✅ Log admin request
	logWithRequest.InfoContext("List transactions request (admin)", map[string]interface{}{
		"page":  page,
		"limit": limit,
	})

	// Call service layer
	resp, err := h.service.ListTransactions(c.Context(), page, limit)
	if err != nil {
		logWithRequest.ErrorContext("Failed to list transactions", err, map[string]interface{}{
			"page":  page,
			"limit": limit,
		})
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  "Failed to fetch transactions",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// GetStatistics handles statistics inquiry (admin endpoint)
func (h *BiFastHandler) GetStatistics(c *fiber.Ctx) error {
	requestID := c.Locals("requestID").(string)
	logWithRequest := h.logger.WithRequestID(requestID)

	logWithRequest.Info("Statistics request (admin)")

	// Call service layer
	stats, err := h.service.GetStatistics(c.Context())
	if err != nil {
		logWithRequest.Error("Failed to fetch statistics", err)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  "Failed to fetch statistics",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(stats)
}

// DeleteTransaction handles transaction deletion (admin endpoint)
func (h *BiFastHandler) DeleteTransaction(c *fiber.Ctx) error {
	requestID := c.Locals("requestID").(string)
	logWithRequest := h.logger.WithRequestID(requestID)

	transactionID := c.Params("transactionId")
	if transactionID == "" {
		logWithRequest.Warn("Transaction ID is required for deletion")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeInvalidRequest,
			ResponseMsg:  "Transaction ID is required",
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	logWithTransfer := logWithRequest.WithTransferID(transactionID)
	logWithTransfer.Warn("Transaction deletion request (admin)")

	// Call service layer
	if err := h.service.DeleteTransaction(c.Context(), transactionID); err != nil {
		logWithTransfer.ErrorContext("Failed to delete transaction", err, nil)

		statusCode := fiber.StatusInternalServerError
		responseCode := dto.ResponseCodeSystemError
		responseMsg := "Failed to delete transaction"

		if err.Error() == "transaction not found" {
			statusCode = fiber.StatusNotFound
			responseCode = dto.ResponseCodeTransactionNotFound
			responseMsg = dto.GetResponseMessage(dto.ResponseCodeTransactionNotFound)
		}

		return c.Status(statusCode).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: responseCode,
			ResponseMsg:  responseMsg,
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	logWithTransfer.Info("Transaction deleted successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"message":   fmt.Sprintf("Transaction %s deleted successfully", transactionID),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ResetAll handles resetting all transactions (admin endpoint)
func (h *BiFastHandler) ResetAll(c *fiber.Ctx) error {
	requestID := c.Locals("requestID").(string)
	logWithRequest := h.logger.WithRequestID(requestID)

	logWithRequest.Warn("Reset all transactions request (admin)")

	// Call service layer
	if err := h.service.ResetAll(c.Context()); err != nil {
		logWithRequest.Error("Failed to reset all transactions", err)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success:      false,
			ResponseCode: dto.ResponseCodeSystemError,
			ResponseMsg:  "Failed to reset all transactions",
			Message:      err.Error(),
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}

	logWithRequest.Info("All transactions reset successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":   true,
		"message":   "All transactions have been reset",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
