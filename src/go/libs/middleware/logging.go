// src/go/libs/middleware/logging.go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// LoggingLogger defines the logger interface for HTTP logging
type LoggingLogger interface {
	Info(msg string, context map[string]interface{})
	Warn(msg string, context map[string]interface{})
	Error(msg string, err error, context map[string]interface{})
}

// LoggingConfig defines logging middleware configuration
type LoggingConfig struct {
	Logger       LoggingLogger
	SkipPaths    []string // Paths to skip logging (e.g., /health)
	LogRequestID bool     // Include request ID in logs
}

// Logging returns an HTTP logging middleware
func Logging(config LoggingConfig) fiber.Handler {
	// Build skip path map for O(1) lookup
	skipMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Skip if path is in skip list
		if skipMap[c.Path()] {
			return c.Next()
		}

		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate response time
		responseTime := time.Since(start)

		// Build log context
		logContext := map[string]interface{}{
			"status":       c.Response().StatusCode(),
			"method":       c.Method(),
			"path":         c.Path(),
			"ip":           c.IP(),
			"user_agent":   c.Get("User-Agent"),
			"responseTime": responseTime.Milliseconds(),
			"bytes":        len(c.Response().Body()),
		}

		// Add request ID if enabled
		if config.LogRequestID {
			if reqID := GetRequestID(c); reqID != "" {
				logContext["requestId"] = reqID
			}
		}

		// Log based on status code
		statusCode := c.Response().StatusCode()
		if err != nil {
			config.Logger.Error("Request error", err, logContext)
		} else if statusCode >= 500 {
			config.Logger.Error("Server error", nil, logContext)
		} else if statusCode >= 400 {
			config.Logger.Warn("Client error", logContext)
		} else {
			config.Logger.Info("Request processed", logContext)
		}

		return err
	}
}

// SimpleLogging returns a basic logging middleware
func SimpleLogging(logger LoggingLogger) fiber.Handler {
	return Logging(LoggingConfig{
		Logger:       logger,
		SkipPaths:    []string{"/health", "/metrics"},
		LogRequestID: true,
	})
}
