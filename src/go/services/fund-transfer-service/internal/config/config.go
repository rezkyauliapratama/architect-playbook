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
	// Existing fields...

	DefaultFlatFee              float64 `mapstructure:"DEFAULT_FLAT_FEE"`
	DefaultPercentageFee        float64 `mapstructure:"DEFAULT_PERCENTAGE_FEE"`
	DefaultDebitAccountCode     string  `mapstructure:"DEFAULT_DEBIT_ACCOUNT_CODE"`
	DefaultCreditAccountCode    string  `mapstructure:"DEFAULT_CREDIT_ACCOUNT_CODE"`
	DefaultFeeDebitAccountCode  string  `mapstructure:"DEFAULT_FEE_DEBIT_ACCOUNT_CODE"`
	DefaultFeeCreditAccountCode string  `mapstructure:"DEFAULT_FEE_CREDIT_ACCOUNT_CODE"`
	DefaultDebitDescription     string  `mapstructure:"DEFAULT_DEBIT_DESCRIPTION"`
	DefaultCreditDescription    string  `mapstructure:"DEFAULT_CREDIT_DESCRIPTION"`
	DefaultFeeDebitDescription  string  `mapstructure:"DEFAULT_FEE_DEBIT_DESCRIPTION"`
	DefaultFeeCreditDescription string  `mapstructure:"DEFAULT_FEE_CREDIT_DESCRIPTION"`
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
