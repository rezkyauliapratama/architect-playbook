// cmd/api/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/middleware"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/handler"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/repository/postgres"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/notification-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger.Initialize(logger.Config{
		LogLevel:     cfg.LogLevel,
		IsProduction: cfg.Environment != "development",
		ServiceName:  "notification-service",
		Version:      "1.0.0",
	})
	log := logger.Get()

	log.InfoContext("Starting Notification Service", map[string]interface{}{
		"environment": cfg.Environment,
		"logLevel":    cfg.LogLevel,
		"port":        cfg.ServerPort,
	})

	// Optimize for CPU usage
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Setup database connection with optimized parameters
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}
	defer db.Close()

	// Configure connection pool for high performance
	db.SetMaxOpenConns(cfg.MaxDBConnections)
	db.SetMaxIdleConns(cfg.MaxDBConnections / 2)
	db.SetConnMaxLifetime(30 * time.Minute)

	log.InfoContext("Database connected", map[string]interface{}{
		"maxConnections": cfg.MaxDBConnections,
	})

	// Initialize dependencies
	apiClient := client.NewAPIClient(cfg)
	emailClient := client.NewEmailClient(cfg)
	notificationRepo := postgres.NewNotificationRepository(db)
	notificationService := service.NewNotificationService(notificationRepo, apiClient, emailClient, cfg)
	notificationHandler := handler.NewNotificationHandler(notificationService)

	// Initialize Fiber app with performance optimizations
	app := fiber.New(fiber.Config{
		Prefork:      false,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    1 * 1024 * 1024, // 1MB limit
		Concurrency:  256 * 1024,      // High concurrency
		ErrorHandler: createErrorHandler(log),
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.LoggingMiddleware())
	// app.Use(middleware.CORS())
	app.Use(compress.New())

	// Set up routes
	api := app.Group("/api/v1")

	notifications := api.Group("/notifications")
	notifications.Post("/", notificationHandler.CreateNotification)
	notifications.Get("/recipient/:recipientId", notificationHandler.GetNotifications)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "UP",
			"service": "notification-service",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Background worker to process pending notifications
	go func() {
		ticker := time.NewTicker(cfg.WorkerInterval)
		defer ticker.Stop()

		log.InfoContext("Background worker started", map[string]interface{}{
			"interval": cfg.WorkerInterval.String(),
		})

		for range ticker.C {
			notificationService.ProcessPendingNotifications() // ✅ FIXED: No error return
		}
	}()

	// Start server
	serverAddr := ":" + cfg.ServerPort
	go func() {
		log.InfoContext("Server starting", map[string]interface{}{
			"address": serverAddr,
		})
		if err := app.Listen(serverAddr); err != nil {
			log.Fatal("Failed to start server", err)
		}
	}()

	log.InfoContext("Server started successfully", map[string]interface{}{
		"port": cfg.ServerPort,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.ErrorContext("Server forced to shutdown", err, nil)
	}

	log.Info("Server stopped gracefully")
}

// createErrorHandler creates a custom error handler for Fiber
func createErrorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		log.ErrorContext("Request error", err, map[string]interface{}{
			"status": code,
			"method": c.Method(),
			"path":   c.Path(),
			"ip":     c.IP(),
		})

		return c.Status(code).JSON(fiber.Map{
			"error":   true,
			"message": err.Error(),
		})
	}
}
