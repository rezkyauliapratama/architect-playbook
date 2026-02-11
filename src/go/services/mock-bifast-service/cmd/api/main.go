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
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/handler"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/middleware"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	setupLogger(cfg.LogLevel)

	log.Info().
		Str("environment", cfg.Environment).
		Str("version", cfg.Version).
		Msg("Starting Mock BI-FAST Service")

	// Initialize database connection
	db, err := initDatabase(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	log.Info().Msg("Database connection established")

	// Initialize repositories
	txnRepo := repository.NewTransactionRepository(db, log.Logger)
	accRepo := repository.NewAccountRepository(log.Logger)

	// Initialize notification client
	notificationClient := client.NewNotificationClient(
		client.NotificationClientConfig{
			BaseURL: cfg.Notification.BaseURL,
			APIKey:  cfg.Notification.APIKey,
			Timeout: cfg.Notification.Timeout,
			Enabled: cfg.Notification.Enabled,
		},
		log.Logger,
	)
	defer notificationClient.Close()

	// Initialize service
	bifastService := service.NewBiFastService(txnRepo, accRepo, notificationClient, log.Logger)

	// Initialize handler
	bifastHandler := handler.NewBiFastHandler(bifastService, log.Logger)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Mock BI-FAST Service",
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		DisableStartupMessage: false,
		ErrorHandler:          middleware.ErrorHandler(log.Logger),
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID,X-Idempotency-Key",
	}))

	// Rate limiter
	app.Use(middleware.RateLimiter(cfg.RateLimit))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "healthy",
			"service":     "mock-bifast-service",
			"version":     cfg.Version,
			"environment": cfg.Environment,
			"timestamp":   time.Now().Format(time.RFC3339),
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Public endpoints
	api.Post("/account-inquiry", bifastHandler.AccountInquiry)
	api.Post("/transfer", bifastHandler.BiFastTransfer)
	api.Get("/transaction/:transactionId", bifastHandler.TransactionStatus)

	// Admin endpoints (protected)
	admin := api.Group("/admin", middleware.AdminAuth(cfg.AdminToken))
	admin.Get("/transactions", bifastHandler.ListTransactions)
	admin.Get("/statistics", bifastHandler.GetStatistics)
	admin.Delete("/transaction/:transactionId", bifastHandler.DeleteTransaction)
	admin.Delete("/reset", bifastHandler.ResetAll)

	// Documentation endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service":     "Mock BI-FAST Service",
			"version":     cfg.Version,
			"description": "Mock implementation of Bank Indonesia FAST payment system for testing and development",
			"endpoints": fiber.Map{
				"health":          "GET /health",
				"accountInquiry":  "POST /api/v1/account-inquiry",
				"transfer":        "POST /api/v1/transfer",
				"transactionInfo": "GET /api/v1/transaction/:transactionId",
				"admin": fiber.Map{
					"listTransactions":  "GET /api/v1/admin/transactions",
					"statistics":        "GET /api/v1/admin/statistics",
					"deleteTransaction": "DELETE /api/v1/admin/transaction/:transactionId",
					"resetAll":          "DELETE /api/v1/admin/reset",
				},
			},
			"documentation": "https://github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service",
		})
	})

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		log.Info().
			Int("port", cfg.Server.Port).
			Str("address", addr).
			Msg("Server starting")

		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited gracefully")
}

func setupLogger(level string) {
	// Set log level
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	// Pretty logging for development
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	})
}

func initDatabase(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
