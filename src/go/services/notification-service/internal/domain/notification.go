// internal/domain/notification.go
package domain

import (
	"time"
)

type NotificationType string
type NotificationStatus string
type NotificationChannel string

const (
	NotificationTypeEmail NotificationType = "EMAIL"
	NotificationTypeSMS   NotificationType = "SMS"
	NotificationTypePush  NotificationType = "PUSH"

	NotificationStatusPending NotificationStatus = "PENDING"
	NotificationStatusSent    NotificationStatus = "SENT"
	NotificationStatusFailed  NotificationStatus = "FAILED"

	// Notification channels - which financial operation triggered it
	ChannelDeposit    NotificationChannel = "DEPOSIT"
	ChannelWithdrawal NotificationChannel = "WITHDRAWAL"
	ChannelBiFast     NotificationChannel = "BI_FAST"
	ChannelRTOL       NotificationChannel = "RTOL"
	ChannelIntrabank  NotificationChannel = "INTRABANK"
	ChannelSystem     NotificationChannel = "SYSTEM"
	ChannelUnknown    NotificationChannel = "UNKNOWN"
)

type Notification struct {
	ID             string                 `json:"id" db:"id"`
	NotificationID string                 `json:"notificationId" db:"notification_id"`
	RecipientID    string                 `json:"recipientId" db:"recipient_id"`
	Type           NotificationType       `json:"type" db:"type"`
	Title          string                 `json:"title" db:"title"`
	Message        string                 `json:"message" db:"message"`
	Status         NotificationStatus     `json:"status" db:"status"`
	Channel        NotificationChannel    `json:"channel" db:"channel"`
	App            string                 `json:"app" db:"app"`
	CreatedAt      time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time              `json:"updatedAt" db:"updated_at"`
	SentAt         *time.Time             `json:"sentAt,omitempty" db:"sent_at"`
	Data           map[string]interface{} `json:"data,omitempty" db:"-"`
}
