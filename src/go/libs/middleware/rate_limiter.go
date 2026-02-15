// src/go/libs/middleware/rate_limiter.go
package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool   // Enable/disable rate limiting
	RequestsPerMinute int    // Max requests per minute per client
	ErrorMessage      string // Custom error message
	ErrorCode         string // Custom error code
}

// RateLimiterStore manages rate limit state
type RateLimiterStore struct {
	mu      sync.RWMutex
	clients map[string]*clientData
}

type clientData struct {
	count     int
	resetTime time.Time
}

// NewRateLimiterStore creates a new rate limiter store with auto-cleanup
func NewRateLimiterStore() *RateLimiterStore {
	store := &RateLimiterStore{
		clients: make(map[string]*clientData),
	}

	// Start cleanup goroutine
	go store.startCleanup()

	return store
}

// startCleanup periodically removes expired entries
func (s *RateLimiterStore) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired client entries
func (s *RateLimiterStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, client := range s.clients {
		if now.After(client.resetTime) {
			delete(s.clients, key)
		}
	}
}

// CheckRateLimit checks if client exceeded rate limit
func (s *RateLimiterStore) CheckRateLimit(clientID string, limit int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Get or create client data
	client, exists := s.clients[clientID]
	if !exists || now.After(client.resetTime) {
		// Create new entry or reset expired entry
		s.clients[clientID] = &clientData{
			count:     1,
			resetTime: now.Add(1 * time.Minute),
		}
		return true
	}

	// Increment counter
	client.count++

	// Check if limit exceeded
	return client.count <= limit
}

// Global store instance
var globalStore = NewRateLimiterStore()

// RateLimiter returns a rate limiting middleware
func RateLimiter(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if disabled
		if !config.Enabled {
			return c.Next()
		}

		// Get client identifier (IP address)
		clientID := c.IP()

		// Check rate limit
		allowed := globalStore.CheckRateLimit(clientID, config.RequestsPerMinute)
		if !allowed {
			errorMsg := config.ErrorMessage
			if errorMsg == "" {
				errorMsg = "Too many requests. Please try again later."
			}

			errorCode := config.ErrorCode
			if errorCode == "" {
				errorCode = "RATE_LIMIT_EXCEEDED"
			}

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Rate limit exceeded",
				"message": errorMsg,
				"code":    errorCode,
			})
		}

		return c.Next()
	}
}

// RateLimiterSimple returns a simple rate limiter with defaults
func RateLimiterSimple(requestsPerMinute int) fiber.Handler {
	return RateLimiter(RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: requestsPerMinute,
	})
}
