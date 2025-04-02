// internal/client/smtp_client.go
package client

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/dto"
)

type EmailClient interface {
	SendEmail(ctx context.Context, req *dto.SendEmailRequest) error
}

type mailcatcherClient struct {
	config *config.Config
}

func NewEmailClient(config *config.Config) EmailClient {
	return &mailcatcherClient{
		config: config,
	}
}

func (c *mailcatcherClient) SendEmail(ctx context.Context, req *dto.SendEmailRequest) error {
	log := logger.Get().WithField("method", "mailcatcherClient.SendEmail")

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Prepare email message
	from := req.From
	if from == "" {
		from = c.config.DefaultFromEmail
	}

	// Build email headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = req.To
	headers["Subject"] = req.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for key, value := range headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	message += "\r\n" + req.HtmlBody

	// Connect to MailCatcher SMTP server
	smtpAddr := fmt.Sprintf("%s:%d", c.config.SmtpHost, c.config.SmtpPort)

	// Create a channel to handle the timeout
	done := make(chan error, 1)
	go func() {
		// Send the email
		err := smtp.SendMail(
			smtpAddr,
			nil, // No authentication needed for MailCatcher
			from,
			[]string{req.To},
			[]byte(message),
		)
		done <- err
	}()

	// Wait for either context timeout or send completion
	select {
	case <-ctx.Done():
		log.Error("SMTP timeout", ctx.Err())
		return fmt.Errorf("SMTP timeout: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			log.Error("Failed to send email", err)
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	log.Info(fmt.Sprint("Email sent successfully", map[string]interface{}{
		"to":      req.To,
		"subject": req.Subject,
	}))

	return nil
}
