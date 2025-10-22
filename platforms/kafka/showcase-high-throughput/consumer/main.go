package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"consumer/config"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Order matches the producer schema
type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotency_key"`
}

// InventoryService manages stock and processes orders
type InventoryService struct {
	consumer  *kafka.Consumer
	inventory map[string]int
	processed map[string]bool // Idempotency tracking
	mu        sync.RWMutex

	// Metrics
	messagesProcessed int64
	messagesSkipped   int64
	messagesFailed    int64
}

// NewInventoryService creates a new service instance
func NewInventoryService(brokers, groupID, instanceID string, topics []string) (*InventoryService, error) {
	// Create consumer with production-grade config
	consumer, err := kafka.NewConsumer(config.ConsumerConfig(brokers, groupID, instanceID))
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Subscribe to topics
	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("✅ Consumer created successfully")
	log.Printf("📋 Configuration:")
	log.Printf("   - Brokers: %s", brokers)
	log.Printf("   - Group ID: %s", groupID)
	log.Printf("   - Instance ID: %s", instanceID)
	log.Printf("   - Topics: %v", topics)
	log.Printf("   - Auto-commit: disabled (manual)")
	log.Printf("   - Isolation: read_committed")

	// Initialize inventory (demo data)
	inventory := map[string]int{
		"prd_laptop_001":   100,
		"prd_mouse_002":    500,
		"prd_keyboard_003": 200,
		"prd_monitor_004":  50,
	}

	log.Printf("📦 Initial inventory:")
	for productID, qty := range inventory {
		log.Printf("   - %s: %d units", productID, qty)
	}

	return &InventoryService{
		consumer:  consumer,
		inventory: inventory,
		processed: make(map[string]bool),
	}, nil
}

// ProcessOrders is the main consumer loop
func (s *InventoryService) ProcessOrders(ctx context.Context) error {
	log.Println("🚀 Inventory Service started, waiting for orders...")

	// Metrics reporting ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹  Shutting down consumer...")
			return s.shutdown()

		case <-ticker.C:
			// Report metrics every 30 seconds
			s.reportMetrics()

		default:
			// Poll for messages with 1 second timeout
			// IMPORTANT: Must poll regularly to send heartbeats
			ev := s.consumer.Poll(1000) // 1 second timeout
			if ev == nil {
				continue // Timeout, continue polling
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Process message
				if err := s.processMessage(e); err != nil {
					log.Printf("❌ Failed to process message: %v", err)
					s.messagesFailed++
					// PRODUCTION: Send to Dead Letter Queue (DLQ)
					continue
				}

				// CRITICAL: Commit offset only after successful processing
				_, err := s.consumer.CommitMessage(e)
				if err != nil {
					log.Printf("⚠️  Failed to commit offset: %v", err)
					// Message will be reprocessed (idempotency ensures safety)
				}

			case kafka.Error:
				// Consumer errors
				log.Printf("❌ Consumer error: %v (code=%v)", e.String(), e.Code())

				// Fatal errors (require restart)
				if e.Code() == kafka.ErrAllBrokersDown {
					return fmt.Errorf("all brokers down")
				}

			case *kafka.Stats:
				// Statistics (every 5 seconds)
				s.handleStats(e)

			default:
				log.Printf("🔍 Unhandled event: %T", ev)
			}
		}
	}
}

func (s *InventoryService) processMessage(msg *kafka.Message) error {
	startTime := time.Now()

	// Parse order
	var order Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	log.Printf("📦 Processing order: %s (partition=%d, offset=%d, key=%s)",
		order.OrderID,
		msg.TopicPartition.Partition,
		msg.TopicPartition.Offset,
		string(msg.Key))

	s.mu.Lock()
	defer s.mu.Unlock()

	// IDEMPOTENCY CHECK: Prevent duplicate processing
	if s.processed[order.IdempotencyKey] {
		log.Printf("⏭  Order %s already processed (idempotent skip)", order.OrderID)
		s.messagesSkipped++
		return nil // Not an error, just skip
	}

	// Check if product exists
	available, exists := s.inventory[order.ProductID]
	if !exists {
		log.Printf("⚠️  Product %s not found in inventory", order.ProductID)
		// PRODUCTION: Send to "orders.rejected" topic
		return nil // Not a fatal error
	}

	// Check inventory availability
	if available < order.Quantity {
		log.Printf("⚠️  Insufficient inventory for %s (need=%d, available=%d)",
			order.ProductID, order.Quantity, available)
		// PRODUCTION: Send to "orders.rejected" topic with reason
		return nil
	}

	// Reserve inventory
	s.inventory[order.ProductID] -= order.Quantity
	s.processed[order.IdempotencyKey] = true
	remaining := s.inventory[order.ProductID]

	processingTime := time.Since(startTime)
	s.messagesProcessed++

	log.Printf("✅ Order %s completed: reserved %d units of %s (remaining=%d, processing_time=%dms)",
		order.OrderID,
		order.Quantity,
		order.ProductID,
		remaining,
		processingTime.Milliseconds())

	// Simulate processing time (remove in production)
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (s *InventoryService) handleStats(stats *kafka.Stats) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stats.String()), &data); err != nil {
		return
	}

	// Extract consumer lag (critical metric)
	if cgrp, ok := data["cgrp"].(map[string]interface{}); ok {
		if assignment, ok := cgrp["assignment"].([]interface{}); ok {
			totalLag := int64(0)
			for _, part := range assignment {
				if p, ok := part.(map[string]interface{}); ok {
					if lag, ok := p["consumer_lag"].(float64); ok {
						totalLag += int64(lag)
					}
				}
			}

			if totalLag > 0 {
				log.Printf("📊 Consumer lag: %d messages", totalLag)

				// PRODUCTION: Alert if lag exceeds threshold
				if totalLag > 10000 {
					log.Printf("⚠️  HIGH LAG ALERT: %d messages behind", totalLag)
				}
			}
		}
	}
}

func (s *InventoryService) reportMetrics() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("📊 Metrics (last 30s):")
	log.Printf("   - Processed: %d", s.messagesProcessed)
	log.Printf("   - Skipped (idempotency): %d", s.messagesSkipped)
	log.Printf("   - Failed: %d", s.messagesFailed)
	log.Printf("   - Unique orders processed: %d", len(s.processed))

	log.Printf("📦 Current inventory:")
	for productID, qty := range s.inventory {
		log.Printf("   - %s: %d units", productID, qty)
	}
}

func (s *InventoryService) shutdown() error {
	log.Println("🛑 Shutting down consumer...")

	// Final metrics report
	s.reportMetrics()

	// Close consumer (commits offsets automatically)
	if err := s.consumer.Close(); err != nil {
		log.Printf("Error closing consumer: %v", err)
		return err
	}

	log.Println("✅ Consumer closed successfully")
	return nil
}

func main() {
	// Read configuration from environment
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:19092" // Default for local testing
	}

	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "inventory-consumer-group"
	}

	instanceID := os.Getenv("HOSTNAME")
	if instanceID == "" {
		instanceID = fmt.Sprintf("inventory-consumer-%d", time.Now().Unix())
	}

	topics := []string{"orders"}

	// Create service
	service, err := NewInventoryService(brokers, groupID, instanceID, topics)
	if err != nil {
		log.Fatal(err)
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle shutdown signals
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigterm
		log.Println("🛑 Shutdown signal received")
		cancel()
	}()

	// Start processing
	if err := service.ProcessOrders(ctx); err != nil {
		log.Fatal(err)
	}
}
