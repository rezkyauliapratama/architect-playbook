package dto

// TransferRequest represents BI-FAST transfer request
type TransferRequest struct {
	// Source information
	SourceBankCode      string `json:"sourceBankCode" validate:"required,len=8"`
	SourceAccountNumber string `json:"sourceAccountNumber" validate:"required,min=1,max=50"`

	// Destination information
	DestBankCode      string `json:"destBankCode" validate:"required,len=8"`
	DestAccountNumber string `json:"destAccountNumber" validate:"required,min=1,max=50"`

	// Transaction details
	Amount      string `json:"amount" validate:"required,numeric"`
	Currency    string `json:"currency" validate:"required,len=3"`
	Description string `json:"description" validate:"max=255"`

	// Reference and idempotency
	ReferenceID    string `json:"referenceId" validate:"required,min=1,max=100"`
	IdempotencyKey string `json:"-"` // Populated from header
}

// TransferResponse represents BI-FAST transfer response
type TransferResponse struct {
	// Response status
	ResponseCode string `json:"responseCode"`
	ResponseMsg  string `json:"responseMsg"`

	// Transaction information
	TransactionID string `json:"transactionId,omitempty"`
	ReferenceID   string `json:"referenceId"`

	// Amount information
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
	Fee      string `json:"fee,omitempty"`

	// Account names
	SourceAccountName string `json:"sourceAccountName,omitempty"`
	DestAccountName   string `json:"destAccountName,omitempty"`

	// Status and timing
	Status          string `json:"status,omitempty"`
	TransactionTime string `json:"transactionTime,omitempty"`
}
