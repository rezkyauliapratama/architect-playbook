// pkg/logger/middleware.go
package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
)

// FiberMiddleware returns a Fiber middleware handler that logs HTTP requests
func FiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get the logger instance
		log := logger.Get()

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

// RateLimiter implements a basic rate limiting middleware
func RateLimiter(max int, duration time.Duration) fiber.Handler {
	// Simple in-memory rate limiter for demonstration
	// In production, consider using Redis or a more robust solution
	type client struct {
		count    int
		lastSeen time.Time
	}

	clients := make(map[string]*client)
	var mu sync.Mutex

	// Cleanup routine to prevent memory leaks
	go func() {
		for {
			time.Sleep(duration)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > duration {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		ip := c.IP()

		mu.Lock()
		if clients[ip] == nil {
			clients[ip] = &client{count: 0, lastSeen: time.Now()}
		}

		clients[ip].lastSeen = time.Now()

		if clients[ip].count >= max {
			mu.Unlock()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests",
				"code":  "BIFAST-E429",
			})
		}

		clients[ip].count++
		mu.Unlock()

		// Start a timer to decrement the counter
		go func() {
			time.Sleep(duration)
			mu.Lock()
			if clients[ip] != nil {
				clients[ip].count--
				if clients[ip].count < 0 {
					clients[ip].count = 0
				}
			}
			mu.Unlock()
		}()

		return c.Next()
	}
}
