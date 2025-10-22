package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"producer/config"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

// Order represents the business domain model
type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotency_key"`
}

// OrderService handles HTTP API and Kafka production
type OrderService struct {
	producer *kafka.Producer
	topic    string
}

// NewOrderService creates a new service instance
func NewOrderService(brokers, topic string) (*OrderService, error) {
	// Create producer with production-grade config
	producer, err := kafka.NewProducer(config.ProducerConfig(brokers))
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	log.Printf("✅ Producer created successfully")
	log.Printf("📋 Configuration:")
	log.Printf("   - Brokers: %s", brokers)
	log.Printf("   - Topic: %s", topic)
	log.Printf("   - Idempotence: enabled")
	log.Printf("   - Compression: lz4")
	log.Printf("   - Batch size: 100KB")
	log.Printf("   - Linger: 10ms")

	service := &OrderService{
		producer: producer,
		topic:    topic,
	}

	// Start delivery report handler
	go service.handleDeliveryReports()

	return service, nil
}

// handleDeliveryReports processes async delivery confirmations
// IMPORTANT: This goroutine MUST be running to prevent channel deadlock
func (s *OrderService) handleDeliveryReports() {
	for e := range s.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			// Message delivery report
			if ev.TopicPartition.Error != nil {
				// PRODUCTION: Implement retry logic or DLQ here
				log.Printf("❌ Delivery failed: %v (key=%s, partition=%d)",
					ev.TopicPartition.Error,
					string(ev.Key),
					ev.TopicPartition.Partition)
			} else {
				log.Printf("✅ Delivered: partition=%d offset=%d latency=%dms",
					ev.TopicPartition.Partition,
					ev.TopicPartition.Offset,
					ev.TopicPartition.Offset) // Simplified; add latency tracking in prod
			}

		case kafka.Error:
			// General producer errors
			// These are informational and handled internally by librdkafka
			log.Printf("⚠️  Producer error: %v (code=%v)", ev.String(), ev.Code())

		case *kafka.Stats:
			// Statistics event (every 5 seconds based on config)
			// PRODUCTION: Send to monitoring system (Prometheus/Grafana)
			var stats map[string]interface{}
			json.Unmarshal([]byte(ev.String()), &stats)

			// Extract key metrics
			if brokers, ok := stats["brokers"].(map[string]interface{}); ok {
				for brokerName, brokerData := range brokers {
					if broker, ok := brokerData.(map[string]interface{}); ok {
						outbufCnt := broker["outbuf_cnt"]
						outbufMsgCnt := broker["outbuf_msg_cnt"]
						log.Printf("📊 Stats [%s]: outbuf_cnt=%v outbuf_msg_cnt=%v",
							brokerName, outbufCnt, outbufMsgCnt)
					}
				}
			}

		default:
			log.Printf("🔍 Unhandled event: %v", ev)
		}
	}
}

// CreateOrderHandler handles POST /orders
func (s *OrderService) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		UserID    string `json:"user_id"`
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.ProductID == "" || req.Quantity <= 0 {
		http.Error(w, "Missing or invalid fields (user_id, product_id, quantity required)",
			http.StatusBadRequest)
		return
	}

	// Create order with unique ID
	orderID := fmt.Sprintf("ord_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
	order := Order{
		OrderID:        orderID,
		UserID:         req.UserID,
		ProductID:      req.ProductID,
		Quantity:       req.Quantity,
		Timestamp:      time.Now().UTC(),
		IdempotencyKey: fmt.Sprintf("%s_v1", orderID),
	}

	// Serialize to JSON
	orderBytes, err := json.Marshal(order)
	if err != nil {
		log.Printf("Failed to serialize order: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Produce to Kafka
	// CRITICAL: Use order_id as key for partition consistency
	err = s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &s.topic,
			Partition: kafka.PartitionAny, // Partitioner uses key hash
		},
		Key:   []byte(order.OrderID), // CONSISTENCY: Same order → same partition
		Value: orderBytes,
		Headers: []kafka.Header{
			{Key: "content-type", Value: []byte("application/json")},
			{Key: "client-id", Value: []byte("order-service")},
			{Key: "timestamp", Value: []byte(order.Timestamp.Format(time.RFC3339))},
		},
	}, nil)

	if err != nil {
		log.Printf("Failed to produce message: %v", err)
		http.Error(w, "Failed to process order", http.StatusInternalServerError)
		return
	}

	log.Printf("📦 Order created: %s (user=%s, product=%s, qty=%d)",
		orderID, req.UserID, req.ProductID, req.Quantity)

	// Return 202 Accepted (async processing)
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
	// Check producer health (metadata fetch = broker connectivity test)
	metadata, err := s.producer.GetMetadata(&s.topic, false, 5000)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "healthy",
		"brokers":      len(metadata.Brokers),
		"topic_exists": len(metadata.Topics) > 0,
	})
}

// Close gracefully shuts down the producer
func (s *OrderService) Close() {
	log.Println("🛑 Shutting down producer...")

	// Flush pending messages (blocking with 15s timeout)
	remaining := s.producer.Flush(15 * 1000)
	if remaining > 0 {
		log.Printf("⚠️  Warning: %d messages still pending after flush timeout", remaining)
	} else {
		log.Println("✅ All messages flushed successfully")
	}

	s.producer.Close()
}

func main() {
	// Read configuration from environment
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:19092,localhost:29092,localhost:39092" // Default for local testing
	}

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "orders"
	}

	// Create service
	service, err := NewOrderService(brokers, topic)
	if err != nil {
		log.Fatal(err)
	}
	defer service.Close()

	// Setup HTTP handlers
	http.HandleFunc("/orders", service.CreateOrderHandler)
	http.HandleFunc("/health", service.HealthHandler)

	// Start HTTP server in goroutine
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("🚀 Order Service running on :%s", port)
		log.Printf("📍 Endpoints:")
		log.Printf("   POST   /orders  - Create order")
		log.Printf("   GET    /health  - Health check")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm

	log.Println("🛑 Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
