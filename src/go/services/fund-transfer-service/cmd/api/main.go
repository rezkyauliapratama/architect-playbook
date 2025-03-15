// cmd/api/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/rezkyauliapratama/architect-playbook/src/go/libs/logger"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/client"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/config"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/handler"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/repository/postgres"
	"github.com/rezkyauliapratama/architect-playbook/src/go/services/fund-transfer-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure database connection pool for optimal performance
	db.SetMaxOpenConns(cfg.MaxDBConnections)
	db.SetMaxIdleConns(cfg.MaxDBConnections / 2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Initialize repositories
	transferRepo := postgres.NewTransferRepository(db)

	// Initialize clients
	accountClient := client.NewAccountClient(cfg)
	transactionClient := client.NewTransactionClient(cfg)
	ledgerClient := client.NewLedgerClient(cfg)

	// Initialize services
	transferService := service.NewTransferService(
		transferRepo,
		accountClient,
		transactionClient,
		ledgerClient,
		cfg,
		cfg.MaxWorkers,
	)

	// Initialize handlers
	transferHandler := handler.NewTransferHandler(transferService)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    1 * 1024 * 1024, // 1MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			// Handle specific error types
			// ...

			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Add middlewares
	app.Use(logger.FiberMiddleware())
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(compress.New()) // Compress responses for better performance

	// Set up routes
	api := app.Group("/api/v1")

	transfers := api.Group("/transfers")
	transfers.Post("/", transferHandler.CreateTransfer)
	transfers.Get("/:transferId", transferHandler.GetTransferByID)
	transfers.Get("/account/:accountId", transferHandler.GetTransfersByAccount)

	bulkTransfers := api.Group("/bulk-transfers")
	bulkTransfers.Post("/", transferHandler.CreateBulkTransfer)
	bulkTransfers.Get("/:bulkTransferId", transferHandler.GetBulkTransferByID)

	// Start the server
	go func() {
		log.Printf("Starting server on %s", cfg.ServerAddress)
		if err := app.Listen(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}
