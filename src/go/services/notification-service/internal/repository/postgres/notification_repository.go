// internal/repository/postgres/notification_repository.go
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/uuid"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/domain"
)

type notificationRepository struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) *notificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	log := logger.Get().WithField("method", "notificationRepository.Create")

	// Generate UUID v7 at code level
	notification.ID = uuid.Generate()

	query := `
        INSERT INTO notifications (
            id, notification_id, recipient_id, type, title, message,
            status, channel, app, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
    `

	_, err := r.db.ExecContext(
		ctx, query,
		notification.ID, notification.NotificationID, notification.RecipientID,
		notification.Type, notification.Title, notification.Message,
		notification.Status, notification.Channel, notification.App,
		notification.CreatedAt, notification.UpdatedAt,
	)

	if err != nil {
		log.Error("Failed to create notification", err)
		return err
	}

	log.Info(fmt.Sprint("Notification created successfully", map[string]interface{}{
		"channel": notification.Channel,
		"app":     notification.App,
	}))
	return nil
}

func (r *notificationRepository) GetByRecipientID(ctx context.Context, recipientID string, channel string, app string, limit, offset int) ([]*domain.Notification, int, error) {
	log := logger.Get().WithField("method", "notificationRepository.GetByRecipientID")

	var query string
	var args []interface{}
	var whereClause string

	// Build dynamic query with proper parameter indexing
	whereClause = "recipient_id = $1"
	args = append(args, recipientID)
	paramIndex := 2

	if channel != "" {
		whereClause += fmt.Sprintf(" AND channel = $%d", paramIndex)
		args = append(args, channel)
		paramIndex++
	}

	if app != "" {
		whereClause += fmt.Sprintf(" AND app = $%d", paramIndex)
		args = append(args, app)
		paramIndex++
	}

	query = fmt.Sprintf(`
        SELECT * FROM notifications 
        WHERE %s
        ORDER BY created_at DESC 
        LIMIT $%d OFFSET $%d
    `, whereClause, paramIndex, paramIndex+1)

	args = append(args, limit, offset)

	var notifications []*domain.Notification
	err := r.db.SelectContext(ctx, &notifications, query, args...)
	if err != nil {
		log.Error("Failed to get notifications", err)
		return nil, 0, err
	}

	// Get total count with same filters
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", whereClause)
	err = r.db.GetContext(ctx, &total, countQuery, args[:paramIndex-1]...)
	if err != nil {
		log.Error("Failed to get total count", err)
		return notifications, 0, err
	}

	return notifications, total, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, notificationID string, status domain.NotificationStatus) error {
	log := logger.Get().WithField("method", "notificationRepository.UpdateStatus")

	query := `
        UPDATE notifications 
        SET status = $1, updated_at = $2
        WHERE notification_id = $3
    `

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, status, now, notificationID)
	if err != nil {
		log.Error("Failed to update status", err)
		return err
	}

	return nil
}

func (r *notificationRepository) UpdateSentTime(ctx context.Context, notificationID string) error {
	log := logger.Get().WithField("method", "notificationRepository.UpdateSentTime")

	query := `
        UPDATE notifications 
        SET sent_at = $1, updated_at = $1, status = $2
        WHERE notification_id = $3
    `

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, domain.NotificationStatusSent, notificationID)
	if err != nil {
		log.Error("Failed to update sent time", err)
		return err
	}

	return nil
}

func (r *notificationRepository) GetPendingNotifications(ctx context.Context, limit int) ([]*domain.Notification, error) {
	log := logger.Get().WithField("method", "notificationRepository.GetPendingNotifications")

	query := `
        SELECT * FROM notifications 
        WHERE status = $1 
        ORDER BY created_at ASC 
        LIMIT $2
        FOR UPDATE SKIP LOCKED
    `

	var notifications []*domain.Notification
	err := r.db.SelectContext(ctx, &notifications, query, domain.NotificationStatusPending, limit)
	if err != nil {
		log.Error("Failed to get pending notifications", err)
		return nil, err
	}

	return notifications, nil
}
