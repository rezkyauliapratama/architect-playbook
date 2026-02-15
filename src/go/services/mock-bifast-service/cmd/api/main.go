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
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/handler"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/repository/postgres"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/mock-bifast-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger.Initialize(logger.Config{LogLevel: cfg.LogLevel, IsProduction: cfg.Environment != "development"})
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
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Initialize dependencies
	notificationClient := client.NewNotificationClient(cfg)
	bifastRepo := postgres.NewBifastRepository(db)
	bifastService := service.NewBifastService(
		bifastRepo,
		notificationClient,
		cfg.BiFastFee,
		cfg.BiFastMaxAmount,
		cfg.BiFastMinAmount,
		cfg.BiFastSuccessRate,
	)
	bifastHandler := handler.NewBifastHandler(bifastService)

	// Initialize Fiber app with performance optimizations
	app := fiber.New(fiber.Config{
		Prefork:               false, // Avoid issues with database connections
		ReadTimeout:           5 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
		BodyLimit:             1 * 1024 * 1024, // 1MB limit
		Concurrency:           256 * 1024,      // High concurrency
		DisableStartupMessage: true,            // Reduce logs
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(compress.New()) // Compress responses for better performance
	app.Use(middleware.FiberMiddleware())
	// Add rate limiting to prevent abuse (20 requests per 10 seconds per IP)
	app.Use(middleware.RateLimiter(20, 10*time.Second))

	// Set up routes
	api := app.Group("/api/v1")
	bifast := api.Group("/bifast")

	// BI-Fast endpoints
	bifast.Post("/inquiry", bifastHandler.AccountInquiry)
	bifast.Post("/transfer", bifastHandler.BifastTransfer)
	bifast.Post("/status", bifastHandler.TransactionStatus)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		// Check database connection
		if err := db.Ping(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "DOWN",
				"error":  "Database connection failed",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "UP",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Start server
	serverShutdown := make(chan struct{})
	go func() {
		if err := app.Listen(":" + cfg.ServerPort); err != nil {
			log.Fatal("Failed to start server", err)
		}
		close(serverShutdown)
	}()

	log.Info(fmt.Sprintf("Server started on port %s", cfg.ServerPort))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("Shutting down server...")
	case <-serverShutdown:
		log.Info("Server stopped unexpectedly")
	}

	// Give outstanding requests 5 seconds to complete
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		log.Error("Error during shutdown", err)
	}

	log.Info("Server gracefully stopped")
}
