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
	logger     *logger.Logger
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
	RecipientID string                 `json:"recipientId"` // ✅ Added to match notification service
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`   // ✅ Added to match notification service
	Message     string                 `json:"message"` // ✅ Added to match notification service
	Channel     string                 `json:"channel"` // ✅ Changed from []string to string
	App         string                 `json:"app"`     // ✅ Added to match notification service
	Data        map[string]interface{} `json:"data,omitempty"`
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

	// Determine notification type and message based on status
	var notifType, title, message string

	switch txn.Status {
	case string(models.StatusCompleted):
		notifType = "EMAIL"
		title = "Transaction Completed"
		message = fmt.Sprintf("Your BI-FAST transfer of %s %s has been completed successfully",
			txn.Currency, txn.Amount)
	case string(models.StatusFailed):
		notifType = "EMAIL"
		title = "Transaction Failed"
		message = fmt.Sprintf("Your BI-FAST transfer of %s %s has failed: %s",
			txn.Currency, txn.Amount, txn.ResponseMsg)
	default:
		notifType = "EMAIL"
		title = "Transaction Initiated"
		message = fmt.Sprintf("Your BI-FAST transfer of %s %s has been initiated",
			txn.Currency, txn.Amount)
	}

	// ✅ Define channels to send to (can be multiple)
	channels := []string{"email"}

	// ✅ Send notification to each channel separately
	for _, channel := range channels {
		payload := NotificationRequest{
			RecipientID: txn.SourceAccountNumber, // ✅ Use source account as recipient ID
			Type:        notifType,
			Title:       title,
			Message:     message,
			Channel:     channel, // ✅ Single string per request
			App:         "bifast",
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
		}

		if txn.CompletedAt != nil {
			payload.Data["completedAt"] = txn.CompletedAt.Format(time.RFC3339)
		}

		// ✅ Send notification for this channel
		if err := c.send(ctx, payload, channel); err != nil {
			// Log error but continue with other channels
			c.logger.WarnContext("Failed to send notification", map[string]interface{}{
				"error":         err.Error(),
				"channel":       channel,
				"transactionId": txn.TransactionID,
			})
		}
	}

	return nil
}

// SendAccountInquiryNotification sends account inquiry notification (optional)
func (c *NotificationClient) SendAccountInquiryNotification(ctx context.Context, accountNo string, data map[string]interface{}) error {
	if !c.enabled {
		return nil
	}

	payload := NotificationRequest{
		RecipientID: accountNo,
		Type:        "EMAIL",
		Title:       "Account Inquiry",
		Message:     "Account inquiry has been performed",
		Channel:     "email", // ✅ Single channel
		App:         "bifast",
		Data:        data,
	}

	return c.send(ctx, payload, "email")
}

// send makes HTTP request to notification service
func (c *NotificationClient) send(ctx context.Context, payload NotificationRequest, channel string) error {
	// Marshal payload
	body, err := json.Marshal(payload)
	if err != nil {
		c.logger.ErrorContext("Failed to marshal notification payload", err, map[string]interface{}{
			"type":    payload.Type,
			"channel": channel,
		})
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// ✅ Log the payload for debugging
	c.logger.DebugContext("Sending notification payload", map[string]interface{}{
		"payload": string(body),
		"channel": channel,
	})

	// Create request
	url := fmt.Sprintf("%s/api/v1/notifications", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		c.logger.ErrorContext("Failed to create notification request", err, map[string]interface{}{
			"url":     url,
			"type":    payload.Type,
			"channel": channel,
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
			"url":     url,
			"type":    payload.Type,
			"channel": channel,
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
			"channel":    channel,
		})
		// Don't return error, just log
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.ErrorContext("Notification service returned error", nil, map[string]interface{}{
			"statusCode": resp.StatusCode,
			"type":       payload.Type,
			"channel":    channel,
			"message":    notifResp.Message,
		})
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, notifResp.Message)
	}

	c.logger.InfoContext("Notification sent successfully", map[string]interface{}{
		"type":           payload.Type,
		"channel":        channel,
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
