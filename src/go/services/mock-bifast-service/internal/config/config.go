package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment  string
	Version      string
	LogLevel     string
	Server       ServerConfig
	Database     DatabaseConfig
	RateLimit    RateLimitConfig
	AdminToken   string
	Notification NotificationConfig
	BiFast       BiFastConfig // NEW: BI-FAST specific configuration
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

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

type RateLimitConfig struct {
	RequestsPerMinute int
	Enabled           bool
}

type NotificationConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// BiFastConfig holds BI-FAST specific configuration
type BiFastConfig struct {
	Fee         float64 // Transfer fee in IDR
	MaxAmount   float64 // Maximum transfer amount in IDR
	MinAmount   float64 // Minimum transfer amount in IDR
	SuccessRate int     // Success rate percentage (0-100) for testing
}

// Load loads configuration from environment variables
func Load() *Config {
	// Load .env file if exists (ignored in production)
	_ = godotenv.Load()

	// Build database config
	dbConfig := DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvAsInt("DB_PORT", 5432),
		Name:            getEnv("DB_NAME", "mockbifast"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "postgres"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxConns:        getEnvAsInt("DB_MAX_CONNS", 25),
		MinConns:        getEnvAsInt("DB_MIN_CONNS", 5),
		MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", "1h"),
		MaxConnIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", "30m"),
	}

	// Build connection URL
	dbConfig.URL = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
		dbConfig.SSLMode,
	)

	// Allow override via DB_URL env
	if dbURL := getEnv("DB_URL", ""); dbURL != "" {
		dbConfig.URL = dbURL
	}

	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "1.0.0"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Server: ServerConfig{
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", "30s"),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", "30s"),
		},
		Database: dbConfig,
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvAsInt("RATE_LIMIT_RPM", 1000),
			Enabled:           getEnvAsBool("RATE_LIMIT_ENABLED", true),
		},
		AdminToken: getEnv("ADMIN_TOKEN", "admin-secret-token-change-in-production"),
		Notification: NotificationConfig{
			Enabled: getEnvAsBool("NOTIFICATION_ENABLED", false),
			BaseURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8082"),
			APIKey:  getEnv("NOTIFICATION_API_KEY", ""),
			Timeout: getEnvAsDuration("NOTIFICATION_TIMEOUT", "10s"),
		},
		// NEW: BI-FAST configuration
		BiFast: BiFastConfig{
			Fee:         getEnvAsFloat("BIFAST_FEE", 2500.0),
			MaxAmount:   getEnvAsFloat("BIFAST_MAX_AMOUNT", 250000000.0),
			MinAmount:   getEnvAsFloat("BIFAST_MIN_AMOUNT", 10000.0),
			SuccessRate: getEnvAsInt("BIFAST_SUCCESS_RATE", 98),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	valueStr := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(valueStr); err == nil {
		return duration
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

// NEW: Helper function to get float environment variables
func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return defaultValue
}
