// src/go/services/mock-bifast-service/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Environment  string
	Version      string
	LogLevel     string
	Server       ServerConfig
	Database     DatabaseConfig
	RateLimit    RateLimitConfig
	AdminToken   string
	Notification NotificationConfig
	BiFast       BiFastConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	URL             string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	Enabled           bool
}

// NotificationConfig holds notification service configuration
type NotificationConfig struct {
	Enabled    bool
	ServiceURL string
	APIKey     string
	Timeout    time.Duration
}

// BiFastConfig holds BI-FAST specific configuration
type BiFastConfig struct {
	Fee         float64
	MaxAmount   float64
	MinAmount   float64
	SuccessRate int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Database configuration
	dbConfig := DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvAsInt("DB_PORT", 5432),
		Name:            getEnv("DB_NAME", "bifast"),
		User:            getEnv("DB_USER", "root"),
		Password:        getEnv("DB_PASSWORD", "password"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxConns:        getEnvAsInt("DB_MAX_CONNS", 25),
		MinConns:        getEnvAsInt("DB_MIN_CONNS", 5),
		MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour),
		MaxConnIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
	}

	// Build connection URL if not provided directly
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		dbURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.Name,
			dbConfig.SSLMode,
		)
	}
	dbConfig.URL = dbURL

	// Build complete config
	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "1.0.0"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		Server: ServerConfig{
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},

		Database: dbConfig,

		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvAsInt("RATE_LIMIT_RPM", 1000),
			Enabled:           getEnvAsBool("RATE_LIMIT_ENABLED", true),
		},

		AdminToken: getEnv("ADMIN_TOKEN", "admin-secret-token-change-in-production"),

		Notification: NotificationConfig{
			Enabled:    getEnvAsBool("NOTIFICATION_ENABLED", false),
			ServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8082"),
			APIKey:     getEnv("NOTIFICATION_API_KEY", ""),
			Timeout:    getEnvAsDuration("NOTIFICATION_TIMEOUT", 10*time.Second),
		},

		BiFast: BiFastConfig{
			Fee:         getEnvAsFloat64("BIFAST_FEE", 2500.0),
			MaxAmount:   getEnvAsFloat64("BIFAST_MAX_AMOUNT", 250000000.0),
			MinAmount:   getEnvAsFloat64("BIFAST_MIN_AMOUNT", 10000.0),
			SuccessRate: getEnvAsInt("BIFAST_SUCCESS_RATE", 98),
		},
	}

	// Validate critical configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server port
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate database
	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}

	// Validate rate limit
	if c.RateLimit.RequestsPerMinute < 1 {
		return fmt.Errorf("rate limit RPM must be at least 1")
	}

	// Validate BI-FAST config
	if c.BiFast.Fee < 0 {
		return fmt.Errorf("bifast fee cannot be negative")
	}
	if c.BiFast.MinAmount < 0 {
		return fmt.Errorf("bifast min amount cannot be negative")
	}
	if c.BiFast.MaxAmount < c.BiFast.MinAmount {
		return fmt.Errorf("bifast max amount must be greater than min amount")
	}
	if c.BiFast.SuccessRate < 0 || c.BiFast.SuccessRate > 100 {
		return fmt.Errorf("bifast success rate must be between 0 and 100")
	}

	return nil
}

// Helper functions for environment variable parsing

// getEnv gets string environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets integer environment variable with default
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsFloat64 gets float64 environment variable with default
func getEnvAsFloat64(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsBool gets boolean environment variable with default
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsDuration gets duration environment variable with default
// Accepts formats like: "30s", "5m", "1h", "1h30m"
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
