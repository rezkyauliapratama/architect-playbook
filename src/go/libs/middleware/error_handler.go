// src/go/libs/middleware/error_handler.go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse defines a standard error response structure
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Code      string `json:"code,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ErrorHandlerLogger defines the logger interface for error handler
type ErrorHandlerLogger interface {
	Error(msg string, err error, context map[string]interface{})
}

// ErrorHandlerConfig defines error handler configuration
type ErrorHandlerConfig struct {
	Logger        ErrorHandlerLogger
	IncludeTrace  bool // Include stack trace in response (dev only)
	CustomHandler func(c *fiber.Ctx, err error, code int) error
}

// ErrorHandler returns a generic error handler middleware
func ErrorHandler(config ErrorHandlerConfig) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Default status code
		code := fiber.StatusInternalServerError

		// Retrieve status code if it's a fiber.Error
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		// Log the error if logger is provided
		if config.Logger != nil {
			config.Logger.Error("Request error", err, map[string]interface{}{
				"statusCode": code,
				"method":     c.Method(),
				"path":       c.Path(),
				"ip":         c.IP(),
				"requestId":  GetRequestID(c),
			})
		}

		// Use custom handler if provided
		if config.CustomHandler != nil {
			return config.CustomHandler(c, err, code)
		}

		// Determine error message and code
		errorMsg, errorCode := mapHTTPError(code)

		// Build response
		response := ErrorResponse{
			Success:   false,
			Error:     errorMsg,
			Message:   err.Error(),
			Code:      errorCode,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		// Send JSON response
		return c.Status(code).JSON(response)
	}
}

// SimpleErrorHandler returns a basic error handler without logging
func SimpleErrorHandler() fiber.ErrorHandler {
	return ErrorHandler(ErrorHandlerConfig{})
}

// mapHTTPError maps HTTP status codes to error messages
func mapHTTPError(code int) (string, string) {
	switch code {
	case fiber.StatusBadRequest:
		return "Bad request", "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "Unauthorized", "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "Forbidden", "FORBIDDEN"
	case fiber.StatusNotFound:
		return "Resource not found", "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "Method not allowed", "METHOD_NOT_ALLOWED"
	case fiber.StatusConflict:
		return "Conflict", "CONFLICT"
	case fiber.StatusTooManyRequests:
		return "Too many requests", "RATE_LIMIT_EXCEEDED"
	case fiber.StatusInternalServerError:
		return "Internal server error", "INTERNAL_ERROR"
	case fiber.StatusServiceUnavailable:
		return "Service unavailable", "SERVICE_UNAVAILABLE"
	case fiber.StatusGatewayTimeout:
		return "Gateway timeout", "GATEWAY_TIMEOUT"
	default:
		return "An error occurred", "ERROR"
	}
}
