// pkg/logger/middleware.go
package logger

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// FiberMiddleware returns a Fiber middleware handler that logs HTTP requests
func FiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get the logger instance
		log := Get()

		// Store the logger in context to make it accessible in handlers
		c.Locals("logger", log)

		// Process request
		err := c.Next()

		// Calculate response time
		responseTime := time.Since(start)

		// Prepare log context
		logContext := map[string]interface{}{
			"status":        c.Response().StatusCode(),
			"method":        c.Method(),
			"path":          c.Path(),
			"ip":            c.IP(),
			"user_agent":    c.Get("User-Agent"),
			"response_time": responseTime.String(),
			"bytes":         len(c.Response().Body()),
		}

		// Log based on status code
		if err != nil {
			log.ErrorContext("Request error", err, logContext)
		} else if c.Response().StatusCode() >= 500 {
			log.ErrorContext("Server error", nil, logContext)
		} else if c.Response().StatusCode() >= 400 {
			log.WarnContext("Client error", logContext)
		} else {
			log.InfoContext("Request processed", logContext)
		}

		return err
	}
}

// GetRequestLogger extracts the logger from the Fiber context
func GetRequestLogger(c *fiber.Ctx) *Logger {
	logger, ok := c.Locals("logger").(*Logger)
	if !ok {
		return Get()
	}
	return logger
}
