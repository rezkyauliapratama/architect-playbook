package dto

import "github.com/gofiber/fiber/v2"

// Response codes following BI-FAST standard
const (
	// Success
	ResponseCodeSuccess = "00"

	// Client errors (4xx)
	ResponseCodeInvalidRequest       = "40"
	ResponseCodeAccountNotFound      = "41"
	ResponseCodeInsufficientBalance  = "42"
	ResponseCodeInvalidAmount        = "43"
	ResponseCodeInvalidTransaction   = "44"
	ResponseCodeDuplicateTransaction = "45"

	// Server errors (5xx)
	ResponseCodeSystemError        = "50"
	ResponseCodeTimeoutError       = "51"
	ResponseCodeServiceUnavailable = "52"

	// Transaction specific
	ResponseCodeTransactionNotFound = "60"
	ResponseCodeTransactionExpired  = "61"
	ResponseCodeTransactionRejected = "62"
)

// Response messages mapping
var responseMessages = map[string]string{
	ResponseCodeSuccess:              "Transaction successful",
	ResponseCodeInvalidRequest:       "Invalid request format",
	ResponseCodeAccountNotFound:      "Account not found",
	ResponseCodeInsufficientBalance:  "Insufficient balance",
	ResponseCodeInvalidAmount:        "Invalid amount",
	ResponseCodeInvalidTransaction:   "Invalid transaction",
	ResponseCodeDuplicateTransaction: "Duplicate transaction",
	ResponseCodeSystemError:          "System error",
	ResponseCodeTimeoutError:         "Transaction timeout",
	ResponseCodeServiceUnavailable:   "Service temporarily unavailable",
	ResponseCodeTransactionNotFound:  "Transaction not found",
	ResponseCodeTransactionExpired:   "Transaction expired",
	ResponseCodeTransactionRejected:  "Transaction rejected",
}

// GetResponseMessage returns the message for a response code
func GetResponseMessage(code string) string {
	if msg, ok := responseMessages[code]; ok {
		return msg
	}
	return "Unknown error"
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	Success      bool   `json:"success"`
	Error        string `json:"error"`
	Message      string `json:"message"`
	ResponseCode string `json:"responseCode,omitempty"`
	Timestamp    string `json:"timestamp"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// DeleteResponse represents delete operation response
type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse represents validation error response
type ValidationErrorResponse struct {
	Success bool              `json:"success"`
	Error   string            `json:"error"`
	Details []ValidationError `json:"details"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err error, message string, code string) *ErrorResponse {
	return &ErrorResponse{
		Success:      false,
		Error:        err.Error(),
		Message:      message,
		ResponseCode: code,
	}
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// HTTPError wraps fiber error with custom message
func HTTPError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Success: false,
		Error:   fiber.ErrBadRequest.Message,
		Message: message,
	})
}
