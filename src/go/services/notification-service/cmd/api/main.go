// cmd/api/main.go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
	logger.Initialize(logger.Config{LogLevel: cfg.LogLevel,
		IsProduction: cfg.Environment != "development"})
	log := logger.Get()

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
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(compress.New()) // Compress responses for better performance
	app.Use(middleware.FiberMiddleware())

	// Set up routes
	api := app.Group("/api/v1")

	notifications := api.Group("/notifications")
	notifications.Post("/", notificationHandler.CreateNotification)
	notifications.Get("/recipient/:recipientId", notificationHandler.GetNotifications)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "UP",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Background worker to process pending notifications
	go func() {
		ticker := time.NewTicker(cfg.WorkerInterval)
		defer ticker.Stop()

		for range ticker.C {
			notificationService.ProcessPendingNotifications()
		}
	}()

	// Start server
	go func() {
		if err := app.Listen(":" + cfg.ServerPort); err != nil {
			log.Fatal("Failed to start server", err)
		}
	}()

	log.Info(fmt.Sprint("Server started on port %s", cfg.ServerPort))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	app.Shutdown()
}
