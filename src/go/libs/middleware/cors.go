// src/go/libs/middleware/cors.go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig defines CORS middleware configuration
type CORSConfig struct {
	AllowOrigins     string // Default: "*"
	AllowMethods     string // Default: "GET,POST,PUT,DELETE,OPTIONS"
	AllowHeaders     string // Default: common headers
	AllowCredentials bool   // Default: true
	ExposeHeaders    string // Default: common headers
	MaxAge           int    // Default: 86400 (24 hours)
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Content-Type,Authorization,X-Request-ID,Idempotency-Key,X-API-Key",
		AllowCredentials: true,
		ExposeHeaders:    "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset",
		MaxAge:           86400,
	}
}

// CORS returns a CORS middleware with default configuration
func CORS() fiber.Handler {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig returns a CORS middleware with custom configuration
func CORSWithConfig(config CORSConfig) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     config.AllowOrigins,
		AllowMethods:     config.AllowMethods,
		AllowHeaders:     config.AllowHeaders,
		AllowCredentials: config.AllowCredentials,
		ExposeHeaders:    config.ExposeHeaders,
		MaxAge:           config.MaxAge,
	})
}

// RestrictiveCORS returns a CORS middleware for production (no wildcard origins)
func RestrictiveCORS(allowedOrigins []string) fiber.Handler {
	originsStr := ""
	for i, origin := range allowedOrigins {
		if i > 0 {
			originsStr += ","
		}
		originsStr += origin
	}

	return cors.New(cors.Config{
		AllowOrigins:     originsStr,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Content-Type,Authorization,X-Request-ID,Idempotency-Key",
		AllowCredentials: true,
		ExposeHeaders:    "X-Request-ID",
		MaxAge:           3600, // 1 hour for production
	})
}
