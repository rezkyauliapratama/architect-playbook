// src/go/services/mock-bifast-service/cmd/api/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/middleware"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/handler"
	localMiddleware "github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/middleware"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

func main() {
	// ✅ FIX: Load configuration with error handling
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// ✅ Initialize logger using libs/logger
	logger.Initialize(logger.Config{
		LogLevel:     cfg.LogLevel,
		IsProduction: cfg.Environment == "production",
		ServiceName:  "mock-bifast-service",
		Version:      cfg.Version,
	})
	log := logger.Get()

	// ✅ Use InfoContext for structured startup logs
	log.InfoContext("Starting Mock BI-FAST Service", map[string]interface{}{
		"environment": cfg.Environment,
		"version":     cfg.Version,
		"port":        cfg.Server.Port,
	})

	// Connect to database
	db, err := connectDatabase(cfg.Database, log)
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}
	defer db.Close()

	// Initialize repositories
	txnRepo := repository.NewTransactionRepository(db, log)
	accRepo := repository.NewAccountRepository(db, log)

	// ✅ FIX: Use ServiceURL instead of BaseURL
	notificationClient := client.NewNotificationClient(client.NotificationClientConfig{
		BaseURL: cfg.Notification.ServiceURL,
		APIKey:  cfg.Notification.APIKey,
		Timeout: cfg.Notification.Timeout,
		Enabled: cfg.Notification.Enabled,
	}, log)

	// Initialize service layer
	bifastService := service.NewBiFastService(
		txnRepo,
		accRepo,
		notificationClient,
		cfg.BiFast,
		log,
	)

	// Initialize handler
	bifastHandler := handler.NewBiFastHandler(bifastService, log)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: createErrorHandler(log),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	})

	// ✅ Setup middleware using libs/middleware
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.LoggingMiddleware())
	app.Use(middleware.RateLimiter(middleware.RateLimitConfig{
		Enabled:           cfg.RateLimit.Enabled,
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
		ErrorMessage:      "Too many requests. Please try again later.",
		ErrorCode:         "BIFAST-E429",
	}))

	// Health check endpoint
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

	// BI-FAST endpoints
	bifast := api.Group("/bifast")
	bifast.Post("/account-inquiry", bifastHandler.AccountInquiry)
	bifast.Post("/transfer", bifastHandler.BiFastTransfer)
	bifast.Get("/transactions/:transactionId", bifastHandler.TransactionStatus)

	// Admin endpoints (protected)
	admin := api.Group("/admin", localMiddleware.AdminAuth(cfg.AdminToken))
	admin.Get("/transactions", bifastHandler.ListTransactions)
	admin.Get("/statistics", bifastHandler.GetStatistics)
	admin.Delete("/transactions/:transactionId", bifastHandler.DeleteTransaction)
	admin.Delete("/transactions", bifastHandler.ResetAll)

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.InfoContext("Server starting", map[string]interface{}{
		"address": serverAddr,
		"fee":     cfg.BiFast.Fee,
	})

	// Start server in goroutine
	go func() {
		if err := app.Listen(serverAddr); err != nil {
			log.Fatal("Failed to start server", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error("Server forced to shutdown", err)
	}

	log.Info("Server stopped")
}

// createErrorHandler creates custom error handler using libs/middleware
func createErrorHandler(log *logger.Logger) fiber.ErrorHandler {
	loggerAdapter := &loggerMiddlewareAdapter{log: log}

	return middleware.ErrorHandler(middleware.ErrorHandlerConfig{
		Logger: loggerAdapter,
		CustomHandler: func(c *fiber.Ctx, err error, code int) error {
			errorMsg, errorCode := mapToServiceError(code)

			return c.Status(code).JSON(fiber.Map{
				"success":       false,
				"error":         errorMsg,
				"message":       err.Error(),
				"response_code": errorCode,
				"timestamp":     time.Now().Format(time.RFC3339),
			})
		},
	})
}

// loggerMiddlewareAdapter adapts libs/logger to middleware.Logger interface
type loggerMiddlewareAdapter struct {
	log *logger.Logger
}

func (l *loggerMiddlewareAdapter) Info(msg string, context map[string]interface{}) {
	l.log.InfoContext(msg, context)
}

func (l *loggerMiddlewareAdapter) Warn(msg string, context map[string]interface{}) {
	l.log.WarnContext(msg, context)
}

func (l *loggerMiddlewareAdapter) Error(msg string, err error, context map[string]interface{}) {
	l.log.ErrorContext(msg, err, context)
}

// mapToServiceError maps HTTP status codes to service error codes
func mapToServiceError(code int) (string, string) {
	switch code {
	case fiber.StatusBadRequest:
		return "Bad request", "BIFAST-E400"
	case fiber.StatusNotFound:
		return "Resource not found", "BIFAST-E404"
	case fiber.StatusUnauthorized:
		return "Unauthorized", "BIFAST-E401"
	case fiber.StatusTooManyRequests:
		return "Too many requests", "BIFAST-E429"
	case fiber.StatusInternalServerError:
		return "Internal server error", "BIFAST-E500"
	case fiber.StatusServiceUnavailable:
		return "Service unavailable", "BIFAST-E503"
	default:
		return "An error occurred", "BIFAST-E000"
	}
}

// connectDatabase establishes database connection pool
func connectDatabase(cfg config.DatabaseConfig, log *logger.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(context.Background()); err != nil {
		return nil, err
	}

	log.InfoContext("Database connection pool initialized", map[string]interface{}{
		"maxConns": cfg.MaxConns,
		"minConns": cfg.MinConns,
	})

	return db, nil
}
