// internal/handler/bifast_handler.go
package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

type BifastHandler struct {
	bifastService *service.BifastService
}

func NewBifastHandler(bifastService *service.BifastService) *BifastHandler {
	return &BifastHandler{
		bifastService: bifastService,
	}
}

// AccountInquiry handles account inquiry requests
func (h *BifastHandler) AccountInquiry(c *fiber.Ctx) error {
	log := logger.Get().WithField("handler", "AccountInquiry")
	start := time.Now()

	var req dto.AccountInquiryRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Invalid request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Invalid request format",
			Code:  "BIFAST-E001",
		})
	}

	// Validate request
	if req.AccountNumber == "" && (req.ProxyType == "" || req.ProxyValue == "") {
		log.Warn("Missing required fields")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Either account number or proxy information (type and value) is required",
			Code:  "BIFAST-E002",
		})
	}

	response, err := h.bifastService.AccountInquiry(c.Context(), &req)
	if err != nil {
		log.Error("Account inquiry failed", err)

		if err.Error() == "account not found" || err.Error() == "account not found in specified bank" {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error: "Account not found",
				Code:  "BIFAST-E003",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to process account inquiry",
			Code:  "BIFAST-E004",
		})
	}

	log.Info(fmt.Sprint("Account inquiry processed successfully", map[string]interface{}{
		"accountNumber": req.AccountNumber,
		"proxyType":     req.ProxyType,
		"latency":       time.Since(start).Milliseconds(),
	}))

	return c.Status(fiber.StatusOK).JSON(response)
}

// BifastTransfer handles BI-Fast transfer requests
func (h *BifastHandler) BifastTransfer(c *fiber.Ctx) error {
	log := logger.Get().WithField("handler", "BifastTransfer")
	start := time.Now()

	var req dto.BifastTransferRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Invalid request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Invalid request format",
			Code:  "BIFAST-E101",
		})
	}

	// Basic validation
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Amount must be greater than zero",
			Code:  "BIFAST-E102",
		})
	}

	if req.SourceAccountNumber == "" || req.SourceBankCode == "" ||
		req.DestinationAccountNumber == "" || req.DestinationBankCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Missing required fields",
			Code:  "BIFAST-E103",
		})
	}

	// Process the transfer
	response, err := h.bifastService.BifastTransfer(c.Context(), &req)
	if err != nil {
		log.Error("BI-Fast transfer failed", err)

		// Handle specific error cases
		if err.Error() == "destination account not found" {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error: "Destination account not found",
				Code:  "BIFAST-E104",
			})
		}

		if err.Error() == "destination bank code does not match account" {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error: "Destination bank code does not match account",
				Code:  "BIFAST-E105",
			})
		}

		if err.Error() == "amount below BI-FAST minimum" {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error: "Amount below BI-FAST minimum limit",
				Code:  "BIFAST-E106",
			})
		}

		if err.Error() == "amount exceeds BI-FAST maximum" {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error: "Amount exceeds BI-FAST maximum limit",
				Code:  "BIFAST-E107",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to process BI-Fast transfer",
			Code:  "BIFAST-E108",
		})
	}

	log.Info(fmt.Sprint("BI-Fast transfer initiated successfully", map[string]interface{}{
		"transactionId": response.TransactionID,
		"amount":        response.Amount,
		"source":        req.SourceAccountNumber,
		"destination":   req.DestinationAccountNumber,
		"latency":       time.Since(start).Milliseconds(),
	}))

	return c.Status(fiber.StatusAccepted).JSON(response)
}

// TransactionStatus handles transaction status requests
func (h *BifastHandler) TransactionStatus(c *fiber.Ctx) error {
	log := logger.Get().WithField("handler", "TransactionStatus")
	start := time.Now()

	var req dto.TransactionStatusRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Invalid request body", err)
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Invalid request format",
			Code:  "BIFAST-E201",
		})
	}

	// Validate request
	if req.TransactionID == "" {
		log.Warn("Missing transaction ID")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error: "Transaction ID is required",
			Code:  "BIFAST-E202",
		})
	}

	response, err := h.bifastService.TransactionStatus(c.Context(), &req)
	if err != nil {
		log.Error("Failed to get transaction status", err)

		if err.Error() == "transaction not found" {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error: "Transaction not found",
				Code:  "BIFAST-E203",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: "Failed to retrieve transaction status",
			Code:  "BIFAST-E204",
		})
	}

	log.Info(fmt.Sprint("Transaction status retrieved successfully", map[string]interface{}{
		"transactionId": req.TransactionID,
		"status":        response.Status,
		"latency":       time.Since(start).Milliseconds(),
	}))

	return c.Status(fiber.StatusOK).JSON(response)
}
