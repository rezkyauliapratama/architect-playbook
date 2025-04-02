// internal/handler/notification_handler.go
package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/service"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

func (h *NotificationHandler) CreateNotification(c *fiber.Ctx) error {
	log := logger.Get().WithField("handler", "CreateNotification")
	start := time.Now()

	var req dto.CreateNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn(fmt.Sprint("Invalid request body", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	notification, err := h.notificationService.CreateNotification(c.Context(), &req)
	if err != nil {
		log.Error("Failed to create notification", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Info(fmt.Sprint("Notification created", map[string]interface{}{
		"latency": time.Since(start).Milliseconds(),
	}))

	return c.Status(fiber.StatusCreated).JSON(notification)
}

func (h *NotificationHandler) GetNotifications(c *fiber.Ctx) error {
	recipientID := c.Params("recipientId")
	log := logger.Get().WithField("handler", "GetNotifications").WithField("recipientId", recipientID)
	start := time.Now()

	// Get channel and app from query parameters
	channel := c.Query("channel", "")
	app := c.Query("app", "")

	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	req := &dto.GetNotificationsRequest{
		RecipientID: recipientID,
		Channel:     channel,
		App:         app,
		Limit:       limit,
		Offset:      offset,
	}

	response, err := h.notificationService.GetNotifications(c.Context(), req)
	if err != nil {
		log.Error("Failed to get notifications", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Info(fmt.Sprint("Notifications retrieved", map[string]interface{}{
		"count":   len(response.Notifications),
		"channel": channel,
		"app":     app,
		"latency": time.Since(start).Milliseconds(),
	}))

	return c.Status(fiber.StatusOK).JSON(response)
}
