// src/go/libs/middleware/request_id.go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID generates or retrieves X-Request-ID header
// Stores it in context for tracing throughout the request lifecycle
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request already has X-Request-ID
		requestID := c.Get("X-Request-ID")

		// Generate new UUID if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in locals for handler access
		c.Locals("requestID", requestID)

		// Set response header
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals("requestID").(string); ok {
		return requestID
	}
	return ""
}
