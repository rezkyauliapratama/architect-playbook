package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type InventoryService struct {
	consumer          *kafka.Consumer
	pool              *pgxpool.Pool
	messagesProcessed int64
	messagesFailed    int64
}

func NewInventoryService(brokers, groupID string, pool *pgxpool.Pool) (*InventoryService, error) {
	// ❌ BASELINE: VULNERABLE CONSUMER CONFIGURATION
	config := &kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          groupID,

		// ❌ VULNERABLE 1: Auto-commit enabled (data loss on crash)
		"enable.auto.commit":      true,
		"auto.commit.interval.ms": 5000, // Commits every 5 seconds

		// ❌ VULNERABLE 2: No fetch optimization (low throughput)
		"fetch.min.bytes": 1, // Fetch immediately, no batching

		// Basic settings
		"auto.offset.reset":     "earliest",
		"session.timeout.ms":    10000,
		"heartbeat.interval.ms": 3000,

		"client.id": "inventory-service-baseline-vulnerable",
	}

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	err = consumer.SubscribeTopics([]string{"orders"}, nil)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Println("✅ Consumer created (BASELINE - VULNERABLE)")
	log.Println("   ❌ Auto-commit: ENABLED (data loss risk)")
	log.Println("   ❌ NO idempotency check")
	log.Println("   ❌ NO transaction (inconsistency risk)")
	log.Println("   → Duplicate messages WILL be processed multiple times")

	return &InventoryService{
		consumer: consumer,
		pool:     pool,
	}, nil
}

func (s *InventoryService) ProcessOrders(ctx context.Context) error {
	log.Println("🚀 Inventory Service (BASELINE - VULNERABLE) started")
	log.Println("   Waiting for messages...")
	log.Println()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return s.shutdown()

		case <-ticker.C:
			s.reportMetrics(ctx)

		default:
			// Poll for messages
			ev := s.consumer.Poll(1000)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				if err := s.processMessage(ctx, e); err != nil {
					log.Printf("❌ Processing failed: %v", err)
					s.messagesFailed++
				}

			case kafka.Error:
				log.Printf("❌ Kafka error: %v", e)
			}
		}
	}
}

func (s *InventoryService) processMessage(ctx context.Context, msg *kafka.Message) error {
	startTime := time.Now()

	// Deserialize
	var order Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	log.Printf("📦 Processing: %s (partition=%d, offset=%d, user=%s)",
		order.OrderID,
		msg.TopicPartition.Partition,
		msg.TopicPartition.Offset,
		order.UserID)

	// ═══════════════════════════════════════════════════════════
	// ❌ BASELINE: NO IDEMPOTENCY CHECK
	// This allows duplicate processing of same message
	// ═══════════════════════════════════════════════════════════

	// ❌ BASELINE: NO TRANSACTION
	// Each operation is separate, can fail independently

	// Step 1: Read current stock (NO LOCK)
	var currentStock, currentReserved int
	err := s.pool.QueryRow(ctx, `
		SELECT stock_quantity, reserved_quantity
		FROM products
		WHERE product_id = $1
	`, order.ProductID).Scan(&currentStock, &currentReserved)

	if err != nil {
		log.Printf("   ⚠️  Product not found: %s", order.ProductID)
		return nil // Skip invalid products
	}

	log.Printf("   Current stock: %d, reserved: %d", currentStock, currentReserved)

	// Step 2: Check availability
	available := currentStock - currentReserved
	if available < order.Quantity {
		log.Printf("   ⚠️  Insufficient stock (available: %d, requested: %d)",
			available, order.Quantity)
		return nil // Skip insufficient stock
	}

	// Step 3: Update reservation (UNCONDITIONALLY)
	newReserved := currentReserved + order.Quantity

	_, err = s.pool.Exec(ctx, `
		UPDATE products
		SET reserved_quantity = $1,
		    updated_at = NOW()
		WHERE product_id = $2
	`, newReserved, order.ProductID)

	if err != nil {
		return fmt.Errorf("update reservation failed: %w", err)
	}

	log.Printf("   Reserved: %d → %d", currentReserved, newReserved)

	// Step 4: Record order (SEPARATE OPERATION - may fail)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO processed_orders (
			idempotency_key,
			order_id,
			user_id,
			product_id,
			quantity,
			status,
			processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, order.IdempotencyKey, order.OrderID, order.UserID,
		order.ProductID, order.Quantity, "completed")

	if err != nil {
		// ❌ CRITICAL BUG: If INSERT fails (e.g., duplicate key),
		// reservation already committed but order not recorded!
		log.Printf("   ⚠️  Order record failed (reservation already done): %v", err)
		// Continue anyway - don't fail the consumer
	}

	// Step 5: Log inventory change
	_, err = s.pool.Exec(ctx, `
		INSERT INTO inventory_logs (
			product_id,
			order_id,
			change_type,
			quantity_change,
			stock_before,
			stock_after,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, order.ProductID, order.OrderID, "reserve",
		-order.Quantity, currentReserved, newReserved)

	if err != nil {
		log.Printf("   ⚠️  Log failed: %v", err)
		// Continue anyway
	}

	s.messagesProcessed++
	processingTime := time.Since(startTime)

	log.Printf("✅ Completed: %s (took %dms)",
		order.OrderID, processingTime.Milliseconds())
	log.Println()

	return nil
}

func (s *InventoryService) reportMetrics(ctx context.Context) {
	log.Println("═══════════════════════════════════════════════")
	log.Printf("📊 Metrics Report")
	log.Printf("   Messages processed:  %d", s.messagesProcessed)
	log.Printf("   Messages failed:     %d", s.messagesFailed)

	// Get inventory stats
	var totalProducts, totalStock, totalReserved, totalAvailable int
	err := s.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(SUM(stock_quantity), 0),
			COALESCE(SUM(reserved_quantity), 0),
			COALESCE(SUM(stock_quantity - reserved_quantity), 0)
		FROM products
	`).Scan(&totalProducts, &totalStock, &totalReserved, &totalAvailable)

	if err != nil {
		log.Printf("   ⚠️  Failed to get stats: %v", err)
	} else {
		log.Printf("📦 Inventory:")
		log.Printf("   Products:      %d", totalProducts)
		log.Printf("   Total stock:   %d", totalStock)
		log.Printf("   Reserved:      %d", totalReserved)
		log.Printf("   Available:     %d", totalAvailable)
	}

	// Get processing stats
	var totalOrders, uniqueUsers int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT user_id)
		FROM processed_orders
	`).Scan(&totalOrders, &uniqueUsers)

	if err != nil {
		log.Printf("   ⚠️  Failed to get order stats: %v", err)
	} else {
		duplicates := totalOrders - uniqueUsers
		log.Printf("📋 Orders:")
		log.Printf("   Total records: %d", totalOrders)
		log.Printf("   Unique users:  %d", uniqueUsers)
		if duplicates > 0 {
			log.Printf("   ❌ DUPLICATES: %d (%.1f%%)",
				duplicates, float64(duplicates)/float64(uniqueUsers)*100)
		}
	}

	log.Println("═══════════════════════════════════════════════")
	log.Println()
}

func (s *InventoryService) shutdown() error {
	log.Println("🛑 Shutting down...")

	// Close consumer (will auto-commit pending offsets)
	if err := s.consumer.Close(); err != nil {
		log.Printf("❌ Consumer close error: %v", err)
	}

	// Close database
	s.pool.Close()

	log.Println("✅ Shutdown complete")
	return nil
}

func main() {
	// Configuration
	brokers := getEnv("KAFKA_BROKERS", "redpanda-0:19092,redpanda-1:29092,redpanda-2:39092")
	groupID := getEnv("KAFKA_GROUP_ID", "inventory-group-baseline")

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "inventory_db")

	// Database connection
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to parse DB config: %v", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("❌ Database ping failed: %v", err)
	}
	log.Println("✅ Database connected")

	// Create service
	service, err := NewInventoryService(brokers, groupID, pool)
	if err != nil {
		log.Fatalf("❌ Failed to create service: %v", err)
	}

	// Signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigterm
		log.Println()
		log.Println("⚠️  Received shutdown signal")
		cancel()
	}()

	// Start processing
	log.Println("════════════════════════════════════════════════════════")
	log.Println("   INVENTORY SERVICE (BASELINE - VULNERABLE)")
	log.Println("════════════════════════════════════════════════════════")
	log.Println()

	if err := service.ProcessOrders(ctx); err != nil {
		log.Fatalf("❌ Service error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
