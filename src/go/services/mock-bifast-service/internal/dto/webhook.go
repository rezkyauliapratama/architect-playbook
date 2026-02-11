package dto

// WebhookNotification represents webhook notification payload
type WebhookNotification struct {
	Type      string                 `json:"type"`
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"signature,omitempty"`
}

// WebhookEventType represents webhook event types
type WebhookEventType string

const (
	WebhookEventTransactionCreated   WebhookEventType = "transaction.created"
	WebhookEventTransactionCompleted WebhookEventType = "transaction.completed"
	WebhookEventTransactionFailed    WebhookEventType = "transaction.failed"
	WebhookEventTransactionExpired   WebhookEventType = "transaction.expired"
	WebhookEventAccountInquiry       WebhookEventType = "account.inquiry"
)
