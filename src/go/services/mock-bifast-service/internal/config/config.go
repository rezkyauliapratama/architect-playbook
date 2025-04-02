// internal/config/config.go
package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort             string  `mapstructure:"SERVER_PORT"`
	DatabaseURL            string  `mapstructure:"DATABASE_URL"`
	Environment            string  `mapstructure:"ENVIRONMENT"`
	LogLevel               string  `mapstructure:"LOG_LEVEL"`
	MaxDBConnections       int     `mapstructure:"MAX_DB_CONNECTIONS"`
	BiFastFee              float64 `mapstructure:"BIFAST_FEE"`
	BiFastMaxAmount        float64 `mapstructure:"BIFAST_MAX_AMOUNT"`
	BiFastMinAmount        float64 `mapstructure:"BIFAST_MIN_AMOUNT"`
	BiFastSuccessRate      int     `mapstructure:"BIFAST_SUCCESS_RATE"`
	NotificationServiceURL string  `mapstructure:"NOTIFICATION_SERVICE_URL"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/mock_bifast?sslmode=disable")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_DB_CONNECTIONS", 20)
	viper.SetDefault("BIFAST_FEE", 2500.0)
	viper.SetDefault("BIFAST_MAX_AMOUNT", 250000000.0)
	viper.SetDefault("BIFAST_MIN_AMOUNT", 10000.0)
	viper.SetDefault("BIFAST_SUCCESS_RATE", 98)
	viper.SetDefault("NOTIFICATION_SERVICE_URL", "http://localhost:8081")

	// Read config file if it exists, but continue if not found
	_ = viper.ReadInConfig()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
