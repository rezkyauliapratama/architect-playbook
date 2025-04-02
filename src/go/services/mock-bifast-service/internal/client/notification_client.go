// internal/client/notification_client.go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
)

type NotificationClient interface {
	SendTransferNotification(ctx context.Context, transaction *dto.TransactionStatusResponse) error
}

type notificationClient struct {
	client *resty.Client
	config *config.Config
}

func NewNotificationClient(config *config.Config) NotificationClient {
	client := resty.New()

	// Optimize for performance
	client.SetTimeout(3 * time.Second)
	client.SetRetryCount(2)
	client.SetRetryWaitTime(100 * time.Millisecond)
	client.SetRetryMaxWaitTime(500 * time.Millisecond)

	return &notificationClient{
		client: client,
		config: config,
	}
}

func (c *notificationClient) SendTransferNotification(ctx context.Context, transaction *dto.TransactionStatusResponse) error {
	// If notification service URL is not configured, skip sending notification
	if c.config.NotificationServiceURL == "" {
		return nil
	}

	log := logger.Get().WithField("method", "notificationClient.SendTransferNotification")

	// Format amounts for display
	amountStr := fmt.Sprintf("Rp %.2f", transaction.Amount)

	var message string
	if transaction.Status == "COMPLETED" {
		message = fmt.Sprintf("Your BI-FAST transfer of %s to account %s has been completed successfully. Reference ID: %s",
			amountStr, transaction.DestinationAccount, transaction.ReferenceID)
	} else if transaction.Status == "FAILED" {
		message = fmt.Sprintf("Your BI-FAST transfer of %s to account %s has failed. Please try again. Reference ID: %s",
			amountStr, transaction.DestinationAccount, transaction.ReferenceID)
	} else {
		message = fmt.Sprintf("Your BI-FAST transfer of %s to account %s is being processed. Reference ID: %s",
			amountStr, transaction.DestinationAccount, transaction.ReferenceID)
	}

	notificationReq := &dto.NotificationRequest{
		RecipientID: "customer", // In real system, this would be the customer's ID
		Type:        "EMAIL",
		Title:       fmt.Sprintf("BI-FAST Transfer %s", transaction.Status),
		Message:     message,
		Channel:     "BI_FAST",
		App:         "BANK_APP",
		Data: map[string]interface{}{
			"transactionId": transaction.TransactionID,
			"amount":        transaction.Amount,
			"status":        transaction.Status,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}

	_, err := c.client.R().
		SetContext(ctx).
		SetBody(notificationReq).
		Post(fmt.Sprintf("%s/api/v1/notifications", c.config.NotificationServiceURL))

	if err != nil {
		log.Error("Failed to send notification", err)
		return err
	}

	log.Info(fmt.Sprint("Notification sent successfully", map[string]interface{}{
		"transactionId": transaction.TransactionID,
		"status":        transaction.Status,
	}))

	return nil
}
