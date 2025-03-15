// internal/handler/transfer_handler.go
package handler

import (
	"fund-transfer-service/internal/dto"
	"fund-transfer-service/internal/service"

	"github.com/gofiber/fiber/v2"
)

type TransferHandler struct {
	transferService *service.TransferService
}

func NewTransferHandler(transferService *service.TransferService) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
	}
}

func (h *TransferHandler) CreateTransfer(c *fiber.Ctx) error {
	var req dto.CreateTransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	resp, err := h.transferService.CreateTransfer(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *TransferHandler) GetTransferByID(c *fiber.Ctx) error {
	transferID := c.Params("transferId")
	if transferID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Transfer ID is required",
		})
	}

	transfer, err := h.transferService.GetTransferByID(c.Context(), transferID)
	if err != nil {
		if err.Error() == "transfer not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(transfer)
}

func (h *TransferHandler) GetTransfersByAccount(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if accountID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Account ID is required",
		})
	}

	limit := c.QueryInt("limit", 10)
	offset := c.QueryInt("offset", 0)

	transfers, err := h.transferService.GetTransfersByAccount(c.Context(), accountID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(transfers)
}

func (h *TransferHandler) CreateBulkTransfer(c *fiber.Ctx) error {
	var req dto.CreateBulkTransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.SourceAccountID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Source account ID is required",
		})
	}

	if len(req.Transfers) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one transfer is required",
		})
	}

	resp, err := h.transferService.CreateBulkTransfer(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *TransferHandler) GetBulkTransferByID(c *fiber.Ctx) error {
	bulkTransferID := c.Params("bulkTransferId")
	if bulkTransferID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bulk transfer ID is required",
		})
	}

	// Implementation details for getting bulk transfer by ID
	// ...

	return c.JSON(fiber.Map{
		"message": "Not implemented yet",
	})
}
