// internal/repository/repository.go
package repository

import (
	"context"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/domain"
)

// NotificationRepository defines the interface for notification repository operations
type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
	GetByRecipientID(ctx context.Context, recipientID string, channel string, app string, limit, offset int) ([]*domain.Notification, int, error)
	UpdateStatus(ctx context.Context, notificationID string, status domain.NotificationStatus) error
	UpdateSentTime(ctx context.Context, notificationID string) error
	GetPendingNotifications(ctx context.Context, limit int) ([]*domain.Notification, error)
}
