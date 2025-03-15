// internal/client/account_client.go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/dto"

	"github.com/go-resty/resty/v2"
)

type AccountClient interface {
	GetAccount(ctx context.Context, accountID string) (*dto.AccountResponse, error)
	UpdateBalance(ctx context.Context, accountID string, req *dto.UpdateAccountBalanceRequest) (*dto.UpdateAccountBalanceResponse, error)
}

type restyAccountClient struct {
	client  *resty.Client
	baseURL string
}

// NewAccountClient creates a new client for interacting with the Account Service
func NewAccountClient(cfg *config.Config) AccountClient {
	client := resty.New()

	// Configure client defaults
	client.SetTimeout(cfg.HTTPClientTimeout)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(100 * time.Millisecond)
	client.SetRetryMaxWaitTime(2 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Accept", "application/json")

	return &restyAccountClient{
		client:  client,
		baseURL: cfg.AccountServiceURL,
	}
}

// GetAccount retrieves account details from the Account Service
func (c *restyAccountClient) GetAccount(ctx context.Context, accountID string) (*dto.AccountResponse, error) {
	var result dto.AccountResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		SetError(&dto.ErrorResponse{}).
		Get(fmt.Sprintf("%s/accounts/%s", c.baseURL, accountID))

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get account: status code %d, response: %s",
			resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// UpdateBalance updates an account's balance
func (c *restyAccountClient) UpdateBalance(
	ctx context.Context,
	accountID string,
	req *dto.UpdateAccountBalanceRequest,
) (*dto.UpdateAccountBalanceResponse, error) {
	var result dto.UpdateAccountBalanceResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&result).
		SetError(&dto.ErrorResponse{}).
		Patch(fmt.Sprintf("%s/accounts/%s/balance", c.baseURL, accountID))

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to update account balance: status code %d, response: %s",
			resp.StatusCode(), resp.String())
	}

	return &result, nil
}
