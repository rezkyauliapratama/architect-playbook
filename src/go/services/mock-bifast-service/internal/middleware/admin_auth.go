package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
)

// AdminAuth middleware validates admin authentication
func AdminAuth(adminToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get authorization header
		authHeader := c.Get("Authorization")

		// Check if header exists
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Success:   false,
				Error:     "Unauthorized",
				Message:   "Authorization header is required",
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Success:   false,
				Error:     "Unauthorized",
				Message:   "Invalid authorization format. Use: Bearer <token>",
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}

		token := parts[1]

		// Validate token
		if token != adminToken {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Success:   false,
				Error:     "Unauthorized",
				Message:   "Invalid admin token",
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}

		// Token is valid, continue to next handler
		return c.Next()
	}
}
