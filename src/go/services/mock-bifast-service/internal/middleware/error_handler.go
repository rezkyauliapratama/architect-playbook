package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
)

// ErrorHandler is a custom error handler for Fiber
func ErrorHandler(logger zerolog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Default status code
		code := fiber.StatusInternalServerError

		// Retrieve status code if it's a fiber.Error
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		// Log the error
		logger.Error().
			Err(err).
			Int("statusCode", code).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Msg("Request error")

		// Determine error message and response code
		var errorMsg string
		var responseCode string

		switch code {
		case fiber.StatusBadRequest:
			errorMsg = "Bad request"
			responseCode = dto.ResponseCodeInvalidRequest
		case fiber.StatusNotFound:
			errorMsg = "Resource not found"
			responseCode = dto.ResponseCodeTransactionNotFound
		case fiber.StatusUnauthorized:
			errorMsg = "Unauthorized"
			responseCode = dto.ResponseCodeInvalidRequest
		case fiber.StatusTooManyRequests:
			errorMsg = "Too many requests"
			responseCode = dto.ResponseCodeServiceUnavailable
		case fiber.StatusInternalServerError:
			errorMsg = "Internal server error"
			responseCode = dto.ResponseCodeSystemError
		case fiber.StatusServiceUnavailable:
			errorMsg = "Service unavailable"
			responseCode = dto.ResponseCodeServiceUnavailable
		default:
			errorMsg = "An error occurred"
			responseCode = dto.ResponseCodeSystemError
		}

		// Send JSON response
		return c.Status(code).JSON(dto.ErrorResponse{
			Success:      false,
			Error:        errorMsg,
			Message:      err.Error(),
			ResponseCode: responseCode,
			Timestamp:    time.Now().Format(time.RFC3339),
		})
	}
}
