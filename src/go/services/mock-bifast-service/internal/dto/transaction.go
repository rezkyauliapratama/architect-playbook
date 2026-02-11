package dto

// TransactionStatusResponse represents transaction status query response
type TransactionStatusResponse struct {
	// Response status
	ResponseCode string `json:"responseCode"`
	ResponseMsg  string `json:"responseMsg"`

	// Transaction identification
	TransactionID string `json:"transactionId,omitempty"`
	ReferenceID   string `json:"referenceId,omitempty"`

	// Source information
	SourceBankCode      string `json:"sourceBankCode,omitempty"`
	SourceAccountNumber string `json:"sourceAccountNumber,omitempty"`

	// Destination information
	DestBankCode      string `json:"destBankCode,omitempty"`
	DestAccountNumber string `json:"destAccountNumber,omitempty"`

	// Transaction details
	Amount      string `json:"amount,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Fee         string `json:"fee,omitempty"`
	Description string `json:"description,omitempty"`

	// Status and timing
	Status          string  `json:"status,omitempty"`
	TransactionTime string  `json:"transactionTime,omitempty"`
	CompletedTime   *string `json:"completedTime,omitempty"`
}

// TransactionItem represents a single transaction in list
type TransactionItem struct {
	TransactionID       string  `json:"transactionId"`
	ReferenceID         string  `json:"referenceId"`
	SourceBankCode      string  `json:"sourceBankCode"`
	SourceAccountNumber string  `json:"sourceAccountNumber"`
	DestBankCode        string  `json:"destBankCode"`
	DestAccountNumber   string  `json:"destAccountNumber"`
	Amount              string  `json:"amount"`
	Currency            string  `json:"currency"`
	Fee                 string  `json:"fee"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"createdAt"`
	CompletedAt         *string `json:"completedAt,omitempty"`
}

// TransactionListResponse represents paginated transaction list
type TransactionListResponse struct {
	Success    bool              `json:"success"`
	Data       []TransactionItem `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}
