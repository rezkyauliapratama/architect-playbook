// internal/client/transaction_client.go
package client

import (
	"context"
	"fmt"
	"time"

	"fund-transfer-service/internal/config"
	"fund-transfer-service/internal/dto"

	"github.com/go-resty/resty/v2"
)

type TransactionClient interface {
	CreateTransaction(ctx context.Context, req *dto.CreateTransactionRequest) (*dto.TransactionResponse, error)
}

type restyTransactionClient struct {
	client  *resty.Client
	baseURL string
}

// NewTransactionClient creates a new client for the Transaction Service
func NewTransactionClient(cfg *config.Config) TransactionClient {
	client := resty.New()

	// Configure client defaults
	client.SetTimeout(cfg.HTTPClientTimeout)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(100 * time.Millisecond)
	client.SetRetryMaxWaitTime(2 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Accept", "application/json")

	return &restyTransactionClient{
		client:  client,
		baseURL: cfg.TransactionServiceURL,
	}
}

// CreateTransaction creates a new transaction record
func (c *restyTransactionClient) CreateTransaction(ctx context.Context, req *dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	var result dto.TransactionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&result).
		SetError(&dto.ErrorResponse{}).
		Post(fmt.Sprintf("%s/transactions", c.baseURL))

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 201 {
		return nil, fmt.Errorf("failed to create transaction: status code %d, response: %s",
			resp.StatusCode(), resp.String())
	}

	return &result, nil
}
