// internal/service/notification_service.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/uuid"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/domain"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/dto"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/repository"
)

type NotificationService struct {
	repo       repository.NotificationRepository
	apiClient  client.APIClient
	smtpClient client.EmailClient
	config     *config.Config
}

func NewNotificationService(repo repository.NotificationRepository, apiClient client.APIClient, smtpClient client.EmailClient, config *config.Config) *NotificationService {
	return &NotificationService{
		repo:       repo,
		apiClient:  apiClient,
		smtpClient: smtpClient,
		config:     config,
	}
}

func (s *NotificationService) CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
	log := logger.Get().WithField("method", "NotificationService.CreateNotification")

	// Generate notification ID
	notificationID := fmt.Sprintf("NOTIF-%s", uuid.Generate()[:13])

	now := time.Now()

	// Set default values if not provided
	channel := req.Channel
	if channel == "" {
		channel = string(domain.ChannelUnknown)
	}

	app := req.App
	if app == "" {
		app = "Unknown"
	}

	// Create notification with channel and app information
	notification := &domain.Notification{
		NotificationID: notificationID,
		RecipientID:    req.RecipientID,
		Type:           domain.NotificationType(req.Type),
		Title:          req.Title,
		Message:        req.Message,
		Channel:        domain.NotificationChannel(channel),
		App:            app,
		Status:         domain.NotificationStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           req.Data,
	}

	err := s.repo.Create(ctx, notification)
	if err != nil {
		log.Error("Failed to create notification", err)
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Process notification asynchronously for better performance
	go s.processNotification(notification)

	log.Info(fmt.Sprint("Notification created successfully", map[string]interface{}{
		"notificationId": notification.NotificationID,
		"type":           notification.Type,
		"channel":        notification.Channel,
		"app":            notification.App,
	}))

	return &dto.NotificationResponse{
		NotificationID: notification.NotificationID,
		RecipientID:    notification.RecipientID,
		Type:           string(notification.Type),
		Title:          notification.Title,
		Message:        notification.Message,
		Channel:        string(notification.Channel),
		App:            notification.App,
		Status:         string(notification.Status),
		CreatedAt:      notification.CreatedAt,
		Data:           notification.Data,
	}, nil
}

func (s *NotificationService) GetNotifications(ctx context.Context, req *dto.GetNotificationsRequest) (*dto.GetNotificationsResponse, error) {
	log := logger.Get().WithField("method", "NotificationService.GetNotifications")

	// Use default values if not provided
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Pass channel and app filters to repository
	notifications, total, err := s.repo.GetByRecipientID(ctx, req.RecipientID, req.Channel, req.App, limit, offset)
	if err != nil {
		log.Error("Failed to get notifications", err)
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	var notificationResponses []dto.NotificationResponse
	for _, notification := range notifications {
		notificationResponses = append(notificationResponses, dto.NotificationResponse{
			NotificationID: notification.NotificationID,
			RecipientID:    notification.RecipientID,
			Type:           string(notification.Type),
			Title:          notification.Title,
			Message:        notification.Message,
			Channel:        string(notification.Channel),
			App:            notification.App,
			Status:         string(notification.Status),
			CreatedAt:      notification.CreatedAt,
			SentAt:         notification.SentAt,
		})
	}

	response := &dto.GetNotificationsResponse{
		Notifications: notificationResponses,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	}

	return response, nil
}

func (s *NotificationService) ProcessPendingNotifications() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log := logger.Get().WithField("method", "NotificationService.ProcessPendingNotifications")

	// Get pending notifications (limited batch size for better performance)
	notifications, err := s.repo.GetPendingNotifications(ctx, 100)
	if err != nil {
		log.Error("Failed to get pending notifications", err)
		return
	}

	if len(notifications) == 0 {
		log.Debug("No pending notifications found")
		return
	}

	log.Info(fmt.Sprint("Processing pending notifications", map[string]interface{}{
		"count": len(notifications),
	}))

	// Process each notification
	for _, notification := range notifications {
		go s.processNotification(notification)
	}
}

func (s *NotificationService) processNotification(notification *domain.Notification) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log := logger.Get().WithFields(map[string]interface{}{
		"method":         "NotificationService.processNotification",
		"notificationId": notification.NotificationID,
		"type":           notification.Type,
	})

	var err error

	switch notification.Type {
	case domain.NotificationTypeEmail:
		err = s.sendEmail(ctx, notification)
	case domain.NotificationTypeSMS:
		err = s.sendSMS(ctx, notification)
	case domain.NotificationTypePush:
		err = s.sendPushNotification(ctx, notification)
	default:
		err = fmt.Errorf("unsupported notification type: %s", notification.Type)
	}

	if err != nil {
		log.Error("Failed to process notification", err)
		s.repo.UpdateStatus(ctx, notification.NotificationID, domain.NotificationStatusFailed)
		return
	}

	// Update notification as sent
	s.repo.UpdateSentTime(ctx, notification.NotificationID)

	log.Info("Notification processed successfully")
}

func (s *NotificationService) sendEmail(ctx context.Context, notification *domain.Notification) error {
	log := logger.Get().WithFields(map[string]interface{}{
		"method":  "NotificationService.sendEmail",
		"channel": notification.Channel,
		"app":     notification.App,
	})

	// Create subject with app and channel context
	subject := fmt.Sprintf("[%s-%s] %s", notification.App, notification.Channel, notification.Title)

	// Create HTML body with app and channel information
	htmlBody := fmt.Sprintf(`
        <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
            <div style="background-color: #f8f9fa; padding: 20px; text-align: center;">
                <h2>%s</h2>
                <div style="margin-top: 10px; font-size: 14px; color: #6c757d;">
                    <span style="display: inline-block; background-color: #e2e3e5; border-radius: 4px; padding: 5px 10px; margin-right: 10px;">
                        Channel: %s
                    </span>
                    <span style="display: inline-block; background-color: #e2e3e5; border-radius: 4px; padding: 5px 10px;">
                        App: %s
                    </span>
                </div>
            </div>
            <div style="padding: 20px;">
                %s
            </div>
            <div style="background-color: #f8f9fa; padding: 10px; text-align: center; font-size: 12px; color: #6c757d;">
                <p>This is an automated message from the financial system. Please do not reply.</p>
                <p>Sent on %s</p>
            </div>
        </div>
    `, notification.Title, notification.Channel, notification.App, notification.Message, time.Now().Format("Monday, January 2, 2006 at 3:04 PM"))

	req := &dto.SendEmailRequest{
		To:        notification.RecipientID,
		Subject:   subject,
		HtmlBody:  htmlBody,
		PlainBody: notification.Message,
	}

	err := s.smtpClient.SendEmail(ctx, req)
	if err != nil {
		log.Error("Failed to send email", err)
		return err
	}

	log.Info(fmt.Sprint("Email sent successfully to MailCatcher", map[string]interface{}{
		"recipient": notification.RecipientID,
		"app":       notification.App,
		"channel":   notification.Channel,
	}))

	return nil
}

func (s *NotificationService) sendSMS(ctx context.Context, notification *domain.Notification) error {
	log := logger.Get().WithField("method", "NotificationService.sendSMS")

	// In a real implementation, you would get the phone number from a user service
	// For this example, we'll use the recipientID as the phone number

	req := &dto.SendSMSRequest{
		To:      notification.RecipientID,
		Message: notification.Message,
	}

	_, err := s.apiClient.SendSMS(ctx, req)
	if err != nil {
		log.Error("Failed to send SMS", err)
		return err
	}

	log.Info("SMS sent successfully")
	return nil
}

func (s *NotificationService) sendPushNotification(ctx context.Context, notification *domain.Notification) error {
	log := logger.Get().WithField("method", "NotificationService.sendPushNotification")

	// In a real implementation, you would get the device token from a user service
	// For this example, we'll use the recipientID as the device token

	req := &dto.SendPushRequest{
		DeviceToken: notification.RecipientID,
		Title:       notification.Title,
		Body:        notification.Message,
		Data:        notification.Data,
	}

	_, err := s.apiClient.SendPush(ctx, req)
	if err != nil {
		log.Error("Failed to send push notification", err)
		return err
	}

	log.Info("Push notification sent successfully")
	return nil
}
