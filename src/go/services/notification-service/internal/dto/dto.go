// internal/dto/dto.go
package dto

import (
	"time"
)

// CreateNotificationRequest represents a request to create a new notification
type CreateNotificationRequest struct {
	RecipientID string                 `json:"recipientId" validate:"required"`
	Type        string                 `json:"type" validate:"required,oneof=EMAIL SMS PUSH"`
	Title       string                 `json:"title" validate:"required"`
	Message     string                 `json:"message" validate:"required"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// NotificationResponse represents the response for a notification
type NotificationResponse struct {
	NotificationID string                 `json:"notificationId"`
	RecipientID    string                 `json:"recipientId"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"createdAt"`
	SentAt         *time.Time             `json:"sentAt,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// GetNotificationsRequest represents a request to get notifications
type GetNotificationsRequest struct {
	RecipientID string `json:"recipientId" validate:"required"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// GetNotificationsResponse represents the response for a list of notifications
type GetNotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int                    `json:"total"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	To      string `json:"to" validate:"required,email"`
	Subject string `json:"subject" validate:"required"`
	Body    string `json:"body" validate:"required"`
}

// SendEmailResponse represents the response from the email service
type SendEmailResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

// SendSMSRequest represents a request to send an SMS
type SendSMSRequest struct {
	To      string `json:"to" validate:"required"`
	Message string `json:"message" validate:"required"`
}

// SendSMSResponse represents the response from the SMS service
type SendSMSResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

// SendPushRequest represents a request to send a push notification
type SendPushRequest struct {
	DeviceToken string                 `json:"deviceToken" validate:"required"`
	Title       string                 `json:"title" validate:"required"`
	Body        string                 `json:"body" validate:"required"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// SendPushResponse represents the response from the push notification service
type SendPushResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}
