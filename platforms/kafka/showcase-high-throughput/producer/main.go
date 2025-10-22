package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

// Order represents the domain model
type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotency_key"`
}

// OrderService handles HTTP requests and Kafka production
type OrderService struct {
	producer *kafka.Producer
	topic    string
}

// NewOrderService creates a new service instance
func NewOrderService(brokers, topic string) (*OrderService, error) {
	// ❌ BASELINE: Naive/Default Kafka producer configuration
	// This demonstrates common mistakes in production
	config := &kafka.ConfigMap{
		"bootstrap.servers": brokers,

		// ❌ MISTAKE 1: Idempotence disabled (default: false)
		// Impact: Network retries can create duplicate messages
		"enable.idempotence": false,

		// ❌ MISTAKE 2: Only wait for leader acknowledgment
		// Impact: Data loss if leader fails before replication
		"acks": 1,

		// ❌ MISTAKE 3: No batching (default values)
		// Impact: One network call per message = terrible throughput
		"batch.size": 16384, // Default 16KB
		"linger.ms":  0,     // Send immediately (no batching)

		// ❌ MISTAKE 4: No compression
		// Impact: High network bandwidth usage
		"compression.type": "none",

		// ❌ MISTAKE 5: Small queue buffer
		// Impact: Backpressure under load
		"queue.buffering.max.messages": 10000, // Small buffer

		// Basic settings
		"client.id":          "order-service-baseline",
		"request.timeout.ms": 30000,
	}

	producer, err := kafka.NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	log.Printf("⚠️  BASELINE Producer created (NOT production-ready)")
	log.Printf("❌ Idempotence: DISABLED (duplicates possible)")
	log.Printf("❌ Acks: 1 (data loss possible)")
	log.Printf("❌ Batching: DISABLED (low throughput)")
	log.Printf("❌ Compression: NONE (high network usage)")

	service := &OrderService{
		producer: producer,
		topic:    topic,
	}

	// Start basic delivery report handler (no error handling)
	go service.handleDeliveryReports()

	return service, nil
}

// handleDeliveryReports - BASELINE: Minimal error handling
func (s *OrderService) handleDeliveryReports() {
	for e := range s.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			// ❌ MISTAKE 6: Just log errors, no retry or DLQ
			if ev.TopicPartition.Error != nil {
				log.Printf("❌ Delivery failed: %v", ev.TopicPartition.Error)
				// In production: Lost message! No retry, no DLQ
			} else {
				log.Printf("✅ Delivered: partition=%d offset=%d",
					ev.TopicPartition.Partition,
					ev.TopicPartition.Offset)
			}
		}
	}
}

// CreateOrderHandler handles POST /orders
func (s *OrderService) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		UserID    string `json:"user_id"`
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.UserID == "" || req.ProductID == "" || req.Quantity <= 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Create order
	orderID := fmt.Sprintf("ord_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
	order := Order{
		OrderID:        orderID,
		UserID:         req.UserID,
		ProductID:      req.ProductID,
		Quantity:       req.Quantity,
		Timestamp:      time.Now().UTC(),
		IdempotencyKey: fmt.Sprintf("%s_v1", orderID),
	}

	// Serialize
	orderBytes, err := json.Marshal(order)
	if err != nil {
		http.Error(w, "Failed to serialize", http.StatusInternalServerError)
		return
	}

	// ❌ MISTAKE 7: NO PARTITION KEY
	// Impact: Messages distributed randomly across partitions
	// Result: Same order can be processed by different consumers simultaneously
	// Consequence: RACE CONDITIONS, out-of-order processing
	err = s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &s.topic,
			Partition: kafka.PartitionAny,
		},
		// ❌ CRITICAL MISTAKE: No key! Random partition assignment
		Key:   nil, // This is the RACE CONDITION source
		Value: orderBytes,
	}, nil)

	if err != nil {
		log.Printf("Failed to produce: %v", err)
		http.Error(w, "Failed to process order", http.StatusInternalServerError)
		return
	}

	log.Printf("📦 Order created: %s (NO KEY - race conditions possible)", orderID)

	// Return 202 Accepted
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":  orderID,
		"status":    "accepted",
		"message":   "Order is being processed",
		"timestamp": order.Timestamp.Format(time.RFC3339),
	})
}

// HealthHandler handles GET /health
func (s *OrderService) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"version": "baseline",
	})
}

// Close - BASELINE: Minimal cleanup
func (s *OrderService) Close() {
	log.Println("🛑 Shutting down baseline producer...")

	// ❌ MISTAKE 8: Short flush timeout
	// Impact: Messages in queue may be lost on shutdown
	remaining := s.producer.Flush(5 * 1000) // Only 5 seconds
	if remaining > 0 {
		log.Printf("⚠️  WARNING: %d messages lost on shutdown", remaining)
	}

	s.producer.Close()
}

func main() {
	// Read configuration
	brokers := "localhost:19092"
	if b := os.Getenv("KAFKA_BROKERS"); b != "" {
		brokers = b
	}

	port := "8081"
	if p := os.Getenv("SERVICE_PORT"); p != "" {
		port = p
	}

	topic := "orders"

	// Create service
	service, err := NewOrderService(brokers, topic)
	if err != nil {
		log.Fatal(err)
	}
	defer service.Close()

	// Setup HTTP handlers
	http.HandleFunc("/orders", service.CreateOrderHandler)
	http.HandleFunc("/health", service.HealthHandler)

	log.Printf("🚀 BASELINE Order Service running on :%s", port)
	log.Printf("⚠️  This is a NAIVE implementation with known issues:")
	log.Printf("   - Race conditions possible (no partition key)")
	log.Printf("   - Low throughput (no batching)")
	log.Printf("   - Duplicate messages possible (no idempotence)")
	log.Printf("   - Data loss possible (acks=1)")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
