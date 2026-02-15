package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/handler"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/middleware"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

// ... (keep existing imports and setup code)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	log := setupLogger(cfg.LogLevel)
	log.Info().
		Str("environment", cfg.Environment).
		Str("version", cfg.Version).
		Int("port", cfg.Server.Port).
		Msg("Starting Mock BI-FAST Service")

	// Connect to database
	db, err := connectDatabase(cfg.Database, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	txnRepo := repository.NewTransactionRepository(db, log)
	accRepo := repository.NewAccountRepository(db, log)

	// Initialize notification client
	notificationClient := client.NewNotificationClient(client.NotificationClientConfig{
		BaseURL: cfg.Notification.BaseURL,
		APIKey:  cfg.Notification.APIKey,
		Timeout: cfg.Notification.Timeout,
		Enabled: cfg.Notification.Enabled,
	}, log)

	// Initialize service with BI-FAST config - UPDATED
	bifastService := service.NewBiFastService(
		txnRepo,
		accRepo,
		notificationClient,
		cfg.BiFast, // NEW: Pass BI-FAST config
		log,
	)

	// Initialize handler
	bifastHandler := handler.NewBiFastHandler(bifastService, log)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(log),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.RateLimiter(cfg.RateLimit))

	// Health check - UPDATED to show config
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "mock-bifast-service",
			"version": cfg.Version,
			"config": fiber.Map{
				"fee":         cfg.BiFast.Fee,
				"maxAmount":   cfg.BiFast.MaxAmount,
				"minAmount":   cfg.BiFast.MinAmount,
				"successRate": cfg.BiFast.SuccessRate,
			},
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// BI-FAST routes
	bifast := api.Group("/bifast")
	bifast.Post("/account-inquiry", bifastHandler.AccountInquiry)
	bifast.Post("/transfer", bifastHandler.BiFastTransfer)
	bifast.Get("/transactions/:transactionId", bifastHandler.TransactionStatus)

	// Admin routes (protected)
	admin := api.Group("/admin", middleware.AdminAuth(cfg.AdminToken))
	admin.Get("/transactions", bifastHandler.ListTransactions)
	admin.Get("/statistics", bifastHandler.GetStatistics)
	admin.Delete("/transactions/:transactionId", bifastHandler.DeleteTransaction)
	admin.Delete("/transactions", bifastHandler.ResetAll)

	// Start server - UPDATED log message
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Info().
		Str("address", serverAddr).
		Float64("fee", cfg.BiFast.Fee).
		Float64("maxAmount", cfg.BiFast.MaxAmount).
		Float64("minAmount", cfg.BiFast.MinAmount).
		Int("successRate", cfg.BiFast.SuccessRate).
		Msg("Server starting with BI-FAST configuration")

	// Graceful shutdown
	go func() {
		if err := app.Listen(serverAddr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

// ... (keep existing setupLogger and connectDatabase functions)

func setupLogger(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var logLevel zerolog.Level
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func connectDatabase(cfg config.DatabaseConfig, log zerolog.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Name).
		Int("maxConns", cfg.MaxConns).
		Msg("Database connected successfully")

	return pool, nil
}
