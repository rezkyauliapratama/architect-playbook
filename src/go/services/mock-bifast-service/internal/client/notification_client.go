// src/go/services/mock-bifast-service/internal/client/notification_client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/models"
)

// NotificationClient handles sending notifications to notification service
type NotificationClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logger.Logger // ✅ Changed from zerolog.Logger
	enabled    bool
}

// NotificationClientConfig holds notification client configuration
type NotificationClientConfig struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	Enabled    bool
	RetryCount int
}

// NotificationRequest represents the request payload to notification service
type NotificationRequest struct {
	Type      string                 `json:"type"`
	Channel   []string               `json:"channel"`
	Recipient NotificationRecipient  `json:"recipient"`
	Data      map[string]interface{} `json:"data"`
	Template  string                 `json:"template,omitempty"`
	Priority  string                 `json:"priority,omitempty"`
}

// NotificationRecipient represents the notification recipient
type NotificationRecipient struct {
	UserID      string `json:"userId,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	AccountNo   string `json:"accountNo,omitempty"`
}

// NotificationResponse represents the response from notification service
type NotificationResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	NotificationID string `json:"notificationId,omitempty"`
}

// NewNotificationClient creates a new notification client
func NewNotificationClient(cfg NotificationClientConfig, log *logger.Logger) *NotificationClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	client := &NotificationClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:  log,
		enabled: cfg.Enabled,
	}

	if cfg.Enabled {
		log.InfoContext("Notification client initialized", map[string]interface{}{
			"baseURL": cfg.BaseURL,
			"timeout": cfg.Timeout.String(),
		})
	} else {
		log.Info("Notification client disabled")
	}

	return client
}

// SendTransactionNotification sends transaction notification
func (c *NotificationClient) SendTransactionNotification(ctx context.Context, txn *models.Transaction) error {
	if !c.enabled {
		c.logger.DebugContext("Notification client disabled, skipping notification", map[string]interface{}{
			"transactionId": txn.TransactionID,
		})
		return nil
	}

	// Determine notification type and template
	notifType := "transaction.created"
	template := "transaction_created"
	priority := "normal"

	switch txn.Status {
	case string(models.StatusCompleted):
		notifType = "transaction.completed"
		template = "transaction_completed"
		priority = "high"
	case string(models.StatusFailed):
		notifType = "transaction.failed"
		template = "transaction_failed"
		priority = "high"
	}

	// Build notification payload
	payload := NotificationRequest{
		Type:    notifType,
		Channel: []string{"push", "email"},
		Recipient: NotificationRecipient{
			AccountNo: txn.SourceAccountNumber,
			// Email and phone would come from account service in production
		},
		Data: map[string]interface{}{
			"transactionId":       txn.TransactionID,
			"referenceId":         txn.ReferenceID,
			"sourceBankCode":      txn.SourceBankCode,
			"sourceAccountNumber": txn.SourceAccountNumber,
			"destBankCode":        txn.DestBankCode,
			"destAccountNumber":   txn.DestAccountNumber,
			"amount":              txn.Amount,
			"currency":            txn.Currency,
			"fee":                 txn.Fee,
			"description":         txn.Description,
			"status":              txn.Status,
			"responseCode":        txn.ResponseCode,
			"responseMsg":         txn.ResponseMsg,
			"createdAt":           txn.CreatedAt.Format(time.RFC3339),
		},
		Template: template,
		Priority: priority,
	}

	if txn.CompletedAt != nil {
		payload.Data["completedAt"] = txn.CompletedAt.Format(time.RFC3339)
	}

	return c.send(ctx, payload)
}

// SendAccountInquiryNotification sends account inquiry notification (optional)
func (c *NotificationClient) SendAccountInquiryNotification(ctx context.Context, data map[string]interface{}) error {
	if !c.enabled {
		return nil
	}

	payload := NotificationRequest{
		Type:     "account.inquiry",
		Channel:  []string{"webhook"},
		Data:     data,
		Priority: "low",
	}

	return c.send(ctx, payload)
}

// send makes HTTP request to notification service
func (c *NotificationClient) send(ctx context.Context, payload NotificationRequest) error {
	// Marshal payload
	body, err := json.Marshal(payload)
	if err != nil {
		c.logger.ErrorContext("Failed to marshal notification payload", err, map[string]interface{}{
			"type": payload.Type,
		})
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/v1/notifications", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		c.logger.ErrorContext("Failed to create notification request", err, map[string]interface{}{
			"url":  url,
			"type": payload.Type,
		})
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MockBiFast/1.0")

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Send request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext("Failed to send notification", err, map[string]interface{}{
			"url":  url,
			"type": payload.Type,
		})
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Parse response
	var notifResp NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&notifResp); err != nil {
		c.logger.WarnContext("Failed to parse notification response", map[string]interface{}{
			"error":      err.Error(),
			"statusCode": resp.StatusCode,
			"type":       payload.Type,
		})
		// Don't return error, just log
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.ErrorContext("Notification service returned error", nil, map[string]interface{}{
			"statusCode": resp.StatusCode,
			"type":       payload.Type,
			"message":    notifResp.Message,
		})
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, notifResp.Message)
	}

	c.logger.InfoContext("Notification sent successfully", map[string]interface{}{
		"type":           payload.Type,
		"notificationId": notifResp.NotificationID,
		"duration":       duration.String(),
		"statusCode":     resp.StatusCode,
	})

	return nil
}

// Close closes the HTTP client connections
func (c *NotificationClient) Close() error {
	c.logger.Info("Closing notification client")
	c.httpClient.CloseIdleConnections()
	return nil
}
