// src/go/libs/middleware/middleware.go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
)

// LoggingMiddleware returns a Fiber middleware that logs HTTP requests
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		log := logger.Get()

		if requestID := GetRequestID(c); requestID != "" {
			log = log.WithRequestID(requestID)
		}

		c.Locals("logger", log)

		err := c.Next()

		responseTime := time.Since(start)

		logContext := map[string]interface{}{
			"status":       c.Response().StatusCode(),
			"method":       c.Method(),
			"path":         c.Path(),
			"ip":           c.IP(),
			"user_agent":   c.Get("User-Agent"),
			"responseTime": responseTime.Milliseconds(),
			"bytes":        len(c.Response().Body()),
		}

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

// GetLoggerFromContext retrieves the logger from Fiber context
func GetLoggerFromContext(c *fiber.Ctx) *logger.Logger {
	if log, ok := c.Locals("logger").(*logger.Logger); ok {
		return log
	}
	return logger.Get()
}
