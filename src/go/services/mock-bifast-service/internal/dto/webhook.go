// src/go/services/mock-bifast-service/internal/dto/webhook.go
package dto

// WebhookNotification represents webhook notification payload
type WebhookNotification struct {
	Type      string                 `json:"type"`                // Event type
	Event     string                 `json:"event"`               // Event name
	Timestamp string                 `json:"timestamp"`           // Event timestamp ISO 8601
	Data      map[string]interface{} `json:"data"`                // Event data payload
	Signature string                 `json:"signature,omitempty"` // HMAC signature untuk verification (optional)
}

// WebhookEventType represents webhook event types
type WebhookEventType string

// Webhook event constants
const (
	WebhookEventTransactionCreated   WebhookEventType = "transaction.created"   // Transaction created
	WebhookEventTransactionCompleted WebhookEventType = "transaction.completed" // Transaction completed successfully
	WebhookEventTransactionFailed    WebhookEventType = "transaction.failed"    // Transaction failed
	WebhookEventTransactionExpired   WebhookEventType = "transaction.expired"   // Transaction expired (TTL)
	WebhookEventAccountInquiry       WebhookEventType = "account.inquiry"       // Account inquiry performed
)
