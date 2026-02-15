// internal/dto/dto.go
package dto

import (
	"time"
)

// Account Inquiry DTOs
type AccountInquiryRequest struct {
	AccountNumber string `json:"accountNumber,omitempty"`
	ProxyType     string `json:"proxyType,omitempty"`  // EMAIL or PHONE
	ProxyValue    string `json:"proxyValue,omitempty"` // Email address or phone number
	BankCode      string `json:"bankCode,omitempty"`
}

type AccountInquiryResponse struct {
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
	BankCode      string `json:"bankCode"`
	BankName      string `json:"bankName"`
	ProxyType     string `json:"proxyType,omitempty"`
	ProxyValue    string `json:"proxyValue,omitempty"`
}

// BI-Fast Transfer DTOs
type BifastTransferRequest struct {
	SourceAccountNumber      string  `json:"sourceAccountNumber" validate:"required"`
	SourceAccountName        string  `json:"sourceAccountName" validate:"required"`
	SourceBankCode           string  `json:"sourceBankCode" validate:"required"`
	DestinationAccountNumber string  `json:"destinationAccountNumber" validate:"required"`
	DestinationAccountName   string  `json:"destinationAccountName,omitempty"`
	DestinationBankCode      string  `json:"destinationBankCode" validate:"required"`
	Amount                   float64 `json:"amount" validate:"required,gt=0"`
	ReferenceID              string  `json:"referenceId" validate:"required"`
	Description              string  `json:"description" validate:"required"`
}

type BifastTransferResponse struct {
	TransactionID      string    `json:"transactionId"`
	Amount             float64   `json:"amount"`
	Fee                float64   `json:"fee"`
	Status             string    `json:"status"`
	ReferenceID        string    `json:"referenceId"`
	SourceAccount      string    `json:"sourceAccount"`
	DestinationAccount string    `json:"destinationAccount"`
	TransactionTime    time.Time `json:"transactionTime"`
}

// Transaction Status DTOs
type TransactionStatusRequest struct {
	TransactionID string `json:"transactionId" validate:"required"`
}

type TransactionStatusResponse struct {
	TransactionID      string     `json:"transactionId"`
	Status             string     `json:"status"`
	Amount             float64    `json:"amount"`
	Fee                float64    `json:"fee"`
	SourceAccount      string     `json:"sourceAccount"`
	DestinationAccount string     `json:"destinationAccount"`
	ReferenceID        string     `json:"referenceId"`
	TransactionTime    time.Time  `json:"transactionTime"`
	CompletedTime      *time.Time `json:"completedTime,omitempty"`
}

// Notification DTOs
type NotificationRequest struct {
	RecipientID string                 `json:"recipientId"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Channel     string                 `json:"channel"`
	App         string                 `json:"app"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Error response
type ErrorResponse struct {
	Error       string `json:"error"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}
