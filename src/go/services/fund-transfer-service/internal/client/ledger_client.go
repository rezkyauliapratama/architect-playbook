// internal/client/ledger_client.go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/dto"

	"github.com/go-resty/resty/v2"
)

type LedgerClient interface {
	CreateEntries(ctx context.Context, req *dto.CreateLedgerEntriesRequest) (*dto.LedgerEntriesResponse, error)
}

type restyLedgerClient struct {
	client  *resty.Client
	baseURL string
}

// NewLedgerClient creates a new client for the Ledger Service
func NewLedgerClient(cfg *config.Config) LedgerClient {
	client := resty.New()

	// Configure client defaults
	client.SetTimeout(cfg.HTTPClientTimeout)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(100 * time.Millisecond)
	client.SetRetryMaxWaitTime(2 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Accept", "application/json")

	return &restyLedgerClient{
		client:  client,
		baseURL: cfg.LedgerServiceURL,
	}
}

// CreateEntries creates ledger entries in the Ledger Service
func (c *restyLedgerClient) CreateEntries(ctx context.Context, req *dto.CreateLedgerEntriesRequest) (*dto.LedgerEntriesResponse, error) {
	var result dto.LedgerEntriesResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&result).
		SetError(&dto.ErrorResponse{}).
		Post(fmt.Sprintf("%s/entries", c.baseURL))

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 201 {
		return nil, fmt.Errorf("failed to create ledger entries: status code %d, response: %s",
			resp.StatusCode(), resp.String())
	}

	return &result, nil
}
