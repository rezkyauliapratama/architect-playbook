// internal/config/config.go
package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort       string        `mapstructure:"SERVER_PORT"`
	DatabaseURL      string        `mapstructure:"DATABASE_URL"`
	Environment      string        `mapstructure:"ENVIRONMENT"`
	LogLevel         string        `mapstructure:"LOG_LEVEL"`
	MaxDBConnections int           `mapstructure:"MAX_DB_CONNECTIONS"`
	EmailServiceURL  string        `mapstructure:"EMAIL_SERVICE_URL"`
	SMSServiceURL    string        `mapstructure:"SMS_SERVICE_URL"`
	PushServiceURL   string        `mapstructure:"PUSH_SERVICE_URL"`
	WorkerInterval   time.Duration `mapstructure:"WORKER_INTERVAL"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/notification_service?sslmode=disable")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_DB_CONNECTIONS", 20)
	viper.SetDefault("EMAIL_SERVICE_URL", "http://localhost:8081")
	viper.SetDefault("SMS_SERVICE_URL", "http://localhost:8082")
	viper.SetDefault("PUSH_SERVICE_URL", "http://localhost:8083")
	viper.SetDefault("WORKER_INTERVAL", 5*time.Second)

	// Read config file if it exists, but continue if not found
	_ = viper.ReadInConfig()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
