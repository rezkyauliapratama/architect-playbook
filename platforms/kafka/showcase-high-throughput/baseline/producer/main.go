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

type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type OrderService struct {
	producer *kafka.Producer
	topic    string
}

func NewOrderService(brokers, topic string) (*OrderService, error) {
	config := &kafka.ConfigMap{
		"bootstrap.servers": brokers,

		// ❌ CRITICAL: Disable idempotence (allows duplicates from retries)
		"enable.idempotence": false,

		// ❌ Only wait for leader (not all replicas)
		"acks": 1,

		// ❌ CRITICAL: Enable retries (will create duplicates without idempotence)
		"retries":          3,
		"retry.backoff.ms": 100,

		// ❌ CRITICAL: Short timeout (increases retry probability)
		"request.timeout.ms":  1000, // 1 second (very short)
		"delivery.timeout.ms": 3000, // 3 seconds total

		// No batching, no compression
		"batch.size":       16384,
		"linger.ms":        0,
		"compression.type": "none",

		"client.id": "order-service-baseline-vulnerable",
	}

	producer, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Producer created (BASELINE - VULNERABLE TO DUPLICATES)")
	log.Println("   ❌ Idempotence: DISABLED")
	log.Println("   ❌ Retries: ENABLED (3 attempts)")
	log.Println("   ❌ Timeout: 1s (short, increases retry chance)")
	log.Println("   → Network issues WILL cause duplicate messages")

	return &OrderService{
		producer: producer,
		topic:    topic,
	}, nil
}

func (s *OrderService) handleDeliveryReports() {
	for e := range s.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Printf("❌ Delivery failed: %v (will retry)", ev.TopicPartition.Error)
			} else {
				log.Printf("✅ Delivered: partition=%d offset=%d key=%s",
					ev.TopicPartition.Partition,
					ev.TopicPartition.Offset,
					string(ev.Key))
			}
		}
	}
}

func (s *OrderService) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	orderID := fmt.Sprintf("ord_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
	order := Order{
		OrderID:        orderID,
		UserID:         req.UserID,
		ProductID:      req.ProductID,
		Quantity:       req.Quantity,
		Timestamp:      time.Now().UTC(),
		IdempotencyKey: fmt.Sprintf("%s_v1", orderID),
	}

	orderBytes, _ := json.Marshal(order)

	// ❌ No partition key (random distribution)
	err := s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &s.topic,
			Partition: kafka.PartitionAny,
		},
		Key:   nil, // No key
		Value: orderBytes,
	}, nil)

	if err != nil {
		log.Printf("❌ Produce failed: %v", err)
		http.Error(w, "Failed to process order", http.StatusInternalServerError)
		return
	}

	// Don't wait for delivery confirmation (fire and forget)
	// This allows request to return even if delivery fails

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":  orderID,
		"status":    "accepted",
		"timestamp": order.Timestamp.Format(time.RFC3339),
	})
}

func (s *OrderService) Close() {
	s.producer.Flush(15 * 1000)
	s.producer.Close()
}

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:19092")
	port := getEnv("SERVICE_PORT", "8081")
	topic := getEnv("KAFKA_TOPIC", "orders")

	service, err := NewOrderService(brokers, topic)
	if err != nil {
		log.Fatal(err)
	}
	defer service.Close()

	go service.handleDeliveryReports()

	http.HandleFunc("/orders", service.CreateOrderHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "version": "baseline-vulnerable"})
	})

	log.Printf("🚀 Order Service (BASELINE) on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
