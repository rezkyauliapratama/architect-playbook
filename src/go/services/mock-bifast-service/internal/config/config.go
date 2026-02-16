// src/go/services/mock-bifast-service/internal/config/config.go
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
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
	BiFast       BiFastConfig
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

type BiFastConfig struct {
	Fee         float64
	MaxAmount   float64
	MinAmount   float64
	SuccessRate int
}

// Load loads configuration using Viper
func Load() *Config {
	v := viper.New()

	// Set config name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/mock-bifast/")

	// Enable environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("BIFAST")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	// Read config file (optional, won't fail if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Errorf("fatal error reading config file: %w", err))
		}
		// Config file not found; using defaults and env vars
	}

	// Build database config
	dbConfig := DatabaseConfig{
		Host:            v.GetString("database.host"),
		Port:            v.GetInt("database.port"),
		Name:            v.GetString("database.name"),
		User:            v.GetString("database.user"),
		Password:        v.GetString("database.password"),
		SSLMode:         v.GetString("database.sslmode"),
		MaxConns:        v.GetInt("database.max_conns"),
		MinConns:        v.GetInt("database.min_conns"),
		MaxConnLifetime: v.GetDuration("database.max_conn_lifetime"),
		MaxConnIdleTime: v.GetDuration("database.max_conn_idle_time"),
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
	if dbURL := v.GetString("database.url"); dbURL != "" {
		dbConfig.URL = dbURL
	}

	return &Config{
		Environment: v.GetString("environment"),
		Version:     v.GetString("version"),
		LogLevel:    v.GetString("log_level"),
		Server: ServerConfig{
			Port:         v.GetInt("server.port"),
			ReadTimeout:  v.GetDuration("server.read_timeout"),
			WriteTimeout: v.GetDuration("server.write_timeout"),
		},
		Database: dbConfig,
		RateLimit: RateLimitConfig{
			RequestsPerMinute: v.GetInt("rate_limit.requests_per_minute"),
			Enabled:           v.GetBool("rate_limit.enabled"),
		},
		AdminToken: v.GetString("admin_token"),
		Notification: NotificationConfig{
			Enabled: v.GetBool("notification.enabled"),
			BaseURL: v.GetString("notification.base_url"),
			APIKey:  v.GetString("notification.api_key"),
			Timeout: v.GetDuration("notification.timeout"),
		},
		BiFast: BiFastConfig{
			Fee:         v.GetFloat64("bifast.fee"),
			MaxAmount:   v.GetFloat64("bifast.max_amount"),
			MinAmount:   v.GetFloat64("bifast.min_amount"),
			SuccessRate: v.GetInt("bifast.success_rate"),
		},
	}
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Application defaults
	v.SetDefault("environment", "development")
	v.SetDefault("version", "1.0.0")
	v.SetDefault("log_level", "info")

	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "mockbifast")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_conns", 25)
	v.SetDefault("database.min_conns", 5)
	v.SetDefault("database.max_conn_lifetime", "1h")
	v.SetDefault("database.max_conn_idle_time", "30m")

	// Rate limit defaults
	v.SetDefault("rate_limit.requests_per_minute", 1000)
	v.SetDefault("rate_limit.enabled", true)

	// Admin defaults
	v.SetDefault("admin_token", "admin-secret-token-change-in-production")

	// Notification defaults
	v.SetDefault("notification.enabled", false)
	v.SetDefault("notification.base_url", "http://localhost:8082")
	v.SetDefault("notification.api_key", "")
	v.SetDefault("notification.timeout", "10s")

	// BI-FAST defaults
	v.SetDefault("bifast.fee", 2500.0)
	v.SetDefault("bifast.max_amount", 250000000.0)
	v.SetDefault("bifast.min_amount", 10000.0)
	v.SetDefault("bifast.success_rate", 98)
}
