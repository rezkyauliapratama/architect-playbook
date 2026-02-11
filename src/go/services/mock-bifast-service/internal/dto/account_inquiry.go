package dto

// AccountInquiryRequest represents account inquiry request
type AccountInquiryRequest struct {
	BankCode      string `json:"bankCode" validate:"required,len=8"`
	AccountNumber string `json:"accountNumber" validate:"required,min=1,max=50"`
	ReferenceID   string `json:"referenceId" validate:"required,min=1,max=100"`
}

// AccountInquiryResponse represents account inquiry response
type AccountInquiryResponse struct {
	ResponseCode  string `json:"responseCode"`
	ResponseMsg   string `json:"responseMsg"`
	ReferenceID   string `json:"referenceId"`
	BankCode      string `json:"bankCode,omitempty"`
	AccountNumber string `json:"accountNumber,omitempty"`
	AccountName   string `json:"accountName,omitempty"`
	AccountType   string `json:"accountType,omitempty"`
}
