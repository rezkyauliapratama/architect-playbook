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
	"github.com/rs/zerolog"

	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/handler"
	localMiddleware "github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/middleware"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"

	// NEW: Import libs middleware
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/middleware"
)

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

	// Initialize service
	bifastService := service.NewBiFastService(
		txnRepo,
		accRepo,
		notificationClient,
		cfg.BiFast,
		log,
	)

	// Initialize handler
	bifastHandler := handler.NewBiFastHandler(bifastService, log)

	// Setup Fiber app with UPDATED error handler from libs
	app := fiber.New(fiber.Config{
		ErrorHandler: createErrorHandler(log),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	})

	// UPDATED: Use libs middleware
	app.Use(recover.New())
	app.Use(middleware.RequestID())                            // FROM LIBS
	app.Use(middleware.CORS())                                 // FROM LIBS
	app.Use(middleware.RateLimiter(middleware.RateLimitConfig{ // FROM LIBS
		Enabled:           cfg.RateLimit.Enabled,
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
		ErrorMessage:      "Too many requests. Please try again later.",
		ErrorCode:         "BIFAST-E429",
	}))

	// Health check
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

	// Admin routes (protected) - STILL USES LOCAL middleware
	admin := api.Group("/admin", localMiddleware.AdminAuth(cfg.AdminToken))
	admin.Get("/transactions", bifastHandler.ListTransactions)
	admin.Get("/statistics", bifastHandler.GetStatistics)
	admin.Delete("/transactions/:transactionId", bifastHandler.DeleteTransaction)
	admin.Delete("/transactions", bifastHandler.ResetAll)

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Info().
		Str("address", serverAddr).
		Float64("fee", cfg.BiFast.Fee).
		Msg("Server starting")

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

// createErrorHandler creates a custom error handler using libs middleware
func createErrorHandler(log zerolog.Logger) fiber.ErrorHandler {
	// Adapter to match libs interface
	logger := &zerologAdapter{log: log}

	return middleware.ErrorHandler(middleware.ErrorHandlerConfig{
		Logger: logger,
		CustomHandler: func(c *fiber.Ctx, err error, code int) error {
			// Map to service-specific response format
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

// zerologAdapter adapts zerolog to libs middleware logger interface
type zerologAdapter struct {
	log zerolog.Logger
}

func (z *zerologAdapter) Info(msg string, context map[string]interface{}) {
	event := z.log.Info()
	for k, v := range context {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (z *zerologAdapter) Warn(msg string, context map[string]interface{}) {
	event := z.log.Warn()
	for k, v := range context {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (z *zerologAdapter) Error(msg string, err error, context map[string]interface{}) {
	event := z.log.Error().Err(err)
	for k, v := range context {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// mapToServiceError maps HTTP codes to service-specific error codes
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

	return zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()
}

func connectDatabase(cfg config.DatabaseConfig, log zerolog.Logger) (*pgxpool.Pool, error) {
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

	log.Info().
		Int("maxConns", cfg.MaxConns).
		Int("minConns", cfg.MinConns).
		Msg("Database connection pool initialized")

	return db, nil
}
