// internal/client/api_client.go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/dto"
)

type APIClient interface {
	SendEmail(ctx context.Context, req *dto.SendEmailRequest) (*dto.SendEmailResponse, error)
	SendSMS(ctx context.Context, req *dto.SendSMSRequest) (*dto.SendSMSResponse, error)
	SendPush(ctx context.Context, req *dto.SendPushRequest) (*dto.SendPushResponse, error)
}

type apiClient struct {
	client *resty.Client
	config *config.Config
}

func NewAPIClient(config *config.Config) APIClient {
	client := resty.New()

	// Performance optimizations for resty client
	client.SetTimeout(5 * time.Second)
	client.SetRetryCount(2)
	client.SetRetryWaitTime(100 * time.Millisecond)
	client.SetRetryMaxWaitTime(1 * time.Second)
	client.SetHeader("Content-Type", "application/json")

	return &apiClient{
		client: client,
		config: config,
	}
}

func (c *apiClient) SendEmail(ctx context.Context, req *dto.SendEmailRequest) (*dto.SendEmailResponse, error) {
	log := logger.Get().WithField("method", "apiClient.SendEmail")

	var response dto.SendEmailResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		Post(fmt.Sprintf("%s/api/email", c.config.EmailServiceURL))

	if err != nil {
		log.Error("Failed to call email service", err)
		return nil, err
	}

	if !resp.IsSuccess() {
		log.Error("Email service error", fmt.Errorf("status: %d", resp.StatusCode()))
		return nil, fmt.Errorf("email service error: %d", resp.StatusCode())
	}

	return &response, nil
}

func (c *apiClient) SendSMS(ctx context.Context, req *dto.SendSMSRequest) (*dto.SendSMSResponse, error) {
	log := logger.Get().WithField("method", "apiClient.SendSMS")

	var response dto.SendSMSResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		Post(fmt.Sprintf("%s/api/sms", c.config.SMSServiceURL))

	if err != nil {
		log.Error("Failed to call SMS service", err)
		return nil, err
	}

	if !resp.IsSuccess() {
		log.Error("SMS service error", fmt.Errorf("status: %d", resp.StatusCode()))
		return nil, fmt.Errorf("SMS service error: %d", resp.StatusCode())
	}

	return &response, nil
}

func (c *apiClient) SendPush(ctx context.Context, req *dto.SendPushRequest) (*dto.SendPushResponse, error) {
	log := logger.Get().WithField("method", "apiClient.SendPush")

	var response dto.SendPushResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		Post(fmt.Sprintf("%s/api/push", c.config.PushServiceURL))

	if err != nil {
		log.Error("Failed to call push notification service", err)
		return nil, err
	}

	if !resp.IsSuccess() {
		log.Error("Push notification service error", fmt.Errorf("status: %d", resp.StatusCode()))
		return nil, fmt.Errorf("push notification service error: %d", resp.StatusCode())
	}

	return &response, nil
}
