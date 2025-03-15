// internal/dto/error_dto.go
package dto

// ErrorResponse represents an error response from an API
type ErrorResponse struct {
	Error       string `json:"error"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}
