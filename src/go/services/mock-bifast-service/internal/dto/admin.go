package dto

// AdminTransactionQuery represents query parameters for admin transaction list
type AdminTransactionQuery struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Status string `query:"status"`
	From   string `query:"from"` // Date filter: from
	To     string `query:"to"`   // Date filter: to
}

// AdminDeleteRequest represents admin delete request
type AdminDeleteRequest struct {
	Confirm bool `json:"confirm"`
}

// ResetAllRequest represents reset all data request
type ResetAllRequest struct {
	Confirm       bool   `json:"confirm"`
	ConfirmPhrase string `json:"confirmPhrase"` // Must be "DELETE ALL DATA"
}

// ResetAllResponse represents reset all data response
type ResetAllResponse struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message"`
	DeletedCount   int      `json:"deletedCount"`
	TransactionIDs []string `json:"transactionIds,omitempty"`
}
