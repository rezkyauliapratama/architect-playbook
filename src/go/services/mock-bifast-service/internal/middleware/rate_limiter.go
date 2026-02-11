package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/dto"
)

// RateLimiter implements a simple in-memory rate limiter
type rateLimiterStore struct {
	mu      sync.RWMutex
	clients map[string]*clientData
}

type clientData struct {
	count     int
	resetTime time.Time
}

var store = &rateLimiterStore{
	clients: make(map[string]*clientData),
}

// Cleanup old entries periodically
func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			store.cleanup()
		}
	}()
}

func (s *rateLimiterStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, client := range s.clients {
		if now.After(client.resetTime) {
			delete(s.clients, key)
		}
	}
}

// RateLimiter middleware limits requests per minute per client
func RateLimiter(cfg config.RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if rate limiting is disabled
		if !cfg.Enabled {
			return c.Next()
		}

		// Get client identifier (IP address)
		clientID := c.IP()

		// Check rate limit
		allowed := checkRateLimit(clientID, cfg.RequestsPerMinute)
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(dto.ErrorResponse{
				Success:      false,
				Error:        "Rate limit exceeded",
				Message:      "Too many requests. Please try again later.",
				ResponseCode: dto.ResponseCodeServiceUnavailable,
				Timestamp:    time.Now().Format(time.RFC3339),
			})
		}

		return c.Next()
	}
}

func checkRateLimit(clientID string, limit int) bool {
	store.mu.Lock()
	defer store.mu.Unlock()

	now := time.Now()

	// Get or create client data
	client, exists := store.clients[clientID]
	if !exists || now.After(client.resetTime) {
		// Create new entry or reset expired entry
		store.clients[clientID] = &clientData{
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
