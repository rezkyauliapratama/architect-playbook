// internal/config/config.go
package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerAddress         string        `mapstructure:"SERVER_ADDRESS"`
	DatabaseURL           string        `mapstructure:"DATABASE_URL"`
	AccountServiceURL     string        `mapstructure:"ACCOUNT_SERVICE_URL"`
	TransactionServiceURL string        `mapstructure:"TRANSACTION_SERVICE_URL"`
	LedgerServiceURL      string        `mapstructure:"LEDGER_SERVICE_URL"`
	HTTPClientTimeout     time.Duration `mapstructure:"HTTP_CLIENT_TIMEOUT"`
	MaxDBConnections      int           `mapstructure:"MAX_DB_CONNECTIONS"`
	MaxWorkers            int           `mapstructure:"MAX_WORKERS"`
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_ADDRESS", ":8082")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/fund_transfer?sslmode=disable")
	viper.SetDefault("ACCOUNT_SERVICE_URL", "http://localhost:8081/api/v1")
	viper.SetDefault("TRANSACTION_SERVICE_URL", "http://localhost:8083/api/v1")
	viper.SetDefault("LEDGER_SERVICE_URL", "http://localhost:8084/api/v1")
	viper.SetDefault("HTTP_CLIENT_TIMEOUT", 5*time.Second)
	viper.SetDefault("MAX_DB_CONNECTIONS", 20)
	viper.SetDefault("MAX_WORKERS", 10)

	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
