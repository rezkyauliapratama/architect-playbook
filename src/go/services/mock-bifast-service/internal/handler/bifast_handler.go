// internal/handler/bifast_handler.go
package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

type BiFastHandler struct {
	service service.BiFastService
	logger  *logger.Logger
}

func NewBiFastHandler(service service.BiFastService, log *logger.Logger) *BiFastHandler {
	return &BiFastHandler{
		service: service,
		logger:  log,
	}
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Check if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Router /health [get]
func (h *BiFastHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(dto.HealthResponse{
		Status:    "ok",
		Service:   "mock-bifast-service",
		Version:   "1.0.0",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// AccountInquiry godoc
// @Summary Account inquiry
// @Description Validate account information before transfer
// @Tags bifast
// @Accept json
// @Produce json
// @Param request body dto.AccountInquiryRequest true "Account inquiry request"
// @Success 200 {object} dto.AccountInquiryResponse
// @Failure 400 {object} dto.ValidationErrorResponse
// @Router /api/v1/bifast/account-inquiry [post]
func (h *BiFastHandler) AccountInquiry(c *fiber.Ctx) error {
	var req dto.AccountInquiryRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.WarnContext("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.ValidationErrorResponse{
			Success: false,
			Error:   "Invalid request format",
			Details: []dto.ValidationError{
				{Field: "body", Message: err.Error()},
			},
		})
	}

	// Validate request
	validationResult := dto.ValidateAccountInquiryRequest(&req)
	if !validationResult.Valid {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ValidationErrorResponse{
			Success: false,
			Error:   "Validation failed",
			Details: validationResult.Errors,
		})
	}

	resp, err := h.service.AccountInquiry(c.Context(), &req)
	if err != nil {
		h.logger.ErrorContext("Account inquiry failed", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Internal server error",
			Message: err.Error(),
		})
	}

	return c.JSON(resp)
}

// BiFastTransfer godoc
// @Summary BI-FAST transfer
// @Description Execute BI-FAST real-time payment transfer
// @Tags bifast
// @Accept json
// @Produce json
// @Param X-Idempotency-Key header string true "Idempotency key"
// @Param request body dto.TransferRequest true "Transfer request"
// @Success 200 {object} dto.TransferResponse
// @Failure 400 {object} dto.ValidationErrorResponse
// @Router /api/v1/bifast/transfer [post]
func (h *BiFastHandler) BiFastTransfer(c *fiber.Ctx) error {
	var req dto.TransferRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.WarnContext("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(dto.ValidationErrorResponse{
			Success: false,
			Error:   "Invalid request format",
			Details: []dto.ValidationError{
				{Field: "body", Message: err.Error()},
			},
		})
	}

	// Get idempotency key from header
	req.IdempotencyKey = c.Get("X-Idempotency-Key")

	// Validate request
	validationResult := dto.ValidateTransferRequest(&req)
	if !validationResult.Valid {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ValidationErrorResponse{
			Success: false,
			Error:   "Validation failed",
			Details: validationResult.Errors,
		})
	}

	resp, err := h.service.BiFastTransfer(c.Context(), &req)
	if err != nil {
		h.logger.ErrorContext("Transfer failed", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Internal server error",
			Message: err.Error(),
		})
	}

	return c.JSON(resp)
}

// GetTransactionStatus godoc
// @Summary Get transaction status
// @Description Retrieve transaction status by ID
// @Tags transactions
// @Produce json
// @Param transactionId path string true "Transaction ID"
// @Success 200 {object} models.Transaction
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/bifast/transactions/{transactionId} [get]
func (h *BiFastHandler) GetTransactionStatus(c *fiber.Ctx) error {
	transactionID := c.Params("transactionId")
	if transactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Transaction ID is required",
		})
	}

	txn, err := h.service.GetTransactionStatus(c.Context(), transactionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
			Success:      false,
			Error:        "Transaction not found",
			ResponseCode: dto.ResponseCodeTransactionNotFound,
		})
	}

	return c.JSON(txn)
}

// ListTransactions godoc
// @Summary List transactions (Admin)
// @Description List all transactions with pagination
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} dto.TransactionListResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/admin/transactions [get]
func (h *BiFastHandler) ListTransactions(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	txns, total, err := h.service.ListTransactions(c.Context(), page, limit)
	if err != nil {
		h.logger.ErrorContext("Failed to list transactions", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve transactions",
		})
	}

	return c.JSON(dto.TransactionListResponse{
		Success: true,
		Data:    txns,
		Pagination: dto.PaginationMeta{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: (total + limit - 1) / limit,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// GetStatistics godoc
// @Summary Get statistics (Admin)
// @Description Get transaction statistics
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.StatisticResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/admin/statistics [get]
func (h *BiFastHandler) GetStatistics(c *fiber.Ctx) error {
	stats, err := h.service.GetStatistics(c.Context())
	if err != nil {
		h.logger.ErrorContext("Failed to get statistics", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve statistics",
		})
	}

	return c.JSON(stats)
}

// DeleteTransaction godoc
// @Summary Delete transaction (Admin)
// @Description Delete a specific transaction
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param transactionId path string true "Transaction ID"
// @Success 200 {object} dto.DeleteResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/admin/transactions/{transactionId} [delete]
func (h *BiFastHandler) DeleteTransaction(c *fiber.Ctx) error {
	transactionID := c.Params("transactionId")
	if transactionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Transaction ID is required",
		})
	}

	if err := h.service.DeleteTransaction(c.Context(), transactionID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Transaction not found",
		})
	}

	return c.JSON(dto.DeleteResponse{
		Success: true,
		Message: "Transaction deleted successfully",
	})
}

// ResetAll godoc
// @Summary Reset all transactions (Admin)
// @Description Delete all transactions
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.DeleteResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/admin/transactions [delete]
func (h *BiFastHandler) ResetAll(c *fiber.Ctx) error {
	if err := h.service.ResetAll(c.Context()); err != nil {
		h.logger.ErrorContext("Failed to reset transactions", err, nil)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Success: false,
			Error:   "Failed to reset transactions",
		})
	}

	return c.JSON(dto.DeleteResponse{
		Success: true,
		Message: "All transactions reset successfully",
	})
}
