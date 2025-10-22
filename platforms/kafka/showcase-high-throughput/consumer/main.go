package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"consumer/db"
	"consumer/repository"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Order matches producer schema
type Order struct {
	OrderID        string    `json:"order_id"`
	UserID         string    `json:"user_id"`
	ProductID      string    `json:"product_id"`
	Quantity       int       `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotency_key"`
}

// InventoryService with pgxpool
type InventoryService struct {
	consumer *kafka.Consumer
	repo     *repository.InventoryRepository
	pool     *pgxpool.Pool

	messagesProcessed int64
	messagesSkipped   int64
	messagesFailed    int64
}

func NewInventoryService(
	brokers, groupID, instanceID string,
	topics []string,
	pool *pgxpool.Pool,
) (*InventoryService, error) {
	config := &kafka.ConfigMap{
		"bootstrap.servers":             brokers,
		"group.id":                      groupID,
		"group.instance.id":             instanceID,
		"enable.auto.commit":            false,
		"isolation.level":               "read_committed",
		"auto.offset.reset":             "earliest",
		"fetch.min.bytes":               1048576,
		"fetch.wait.max.ms":             100,
		"session.timeout.ms":            30000,
		"heartbeat.interval.ms":         3000,
		"max.poll.interval.ms":          300000,
		"partition.assignment.strategy": "range",
	}

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}

	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		consumer.Close()
		return nil, err
	}

	log.Printf("✅ Consumer created with pgx backend")

	return &InventoryService{
		consumer: consumer,
		repo:     repository.NewInventoryRepository(pool),
		pool:     pool,
	}, nil
}

func (s *InventoryService) ProcessOrders(ctx context.Context) error {
	log.Println("🚀 Inventory Service started (pgx backend)")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return s.shutdown()
		case <-ticker.C:
			s.reportMetrics(ctx)
		default:
			ev := s.consumer.Poll(1000)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				if err := s.processMessage(ctx, e); err != nil {
					log.Printf("❌ Failed: %v", err)
					s.messagesFailed++
					continue
				}
				s.consumer.CommitMessage(e)
			case kafka.Error:
				log.Printf("❌ Error: %v", e)
			}
		}
	}
}

func (s *InventoryService) processMessage(ctx context.Context, msg *kafka.Message) error {
	var order Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return err
	}

	log.Printf("📦 Processing: %s", order.OrderID)

	existing, err := s.repo.CheckIdempotency(ctx, order.IdempotencyKey)
	if err != nil {
		return err
	}

	if existing != nil {
		log.Printf("⏭  Already processed: %s", order.OrderID)
		s.messagesSkipped++
		return nil
	}

	err = s.repo.ReserveInventory(ctx, order.OrderID, order.UserID, order.ProductID, order.Quantity, order.IdempotencyKey)
	if err != nil {
		if err == repository.ErrInsufficientStock {
			log.Printf("⚠️  Insufficient stock")
			return nil
		}
		return err
	}

	s.messagesProcessed++
	log.Printf("✅ Completed: %s", order.OrderID)
	return nil
}

func (s *InventoryService) reportMetrics(ctx context.Context) {
	stats, _ := s.repo.GetInventoryStats(ctx)
	log.Printf("📊 Processed=%d Skipped=%d Failed=%d",
		s.messagesProcessed, s.messagesSkipped, s.messagesFailed)
	log.Printf("📦 Inventory: %v", stats)
}

func (s *InventoryService) shutdown() error {
	s.pool.Close()
	return s.consumer.Close()
}

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:19092")
	groupID := getEnv("KAFKA_GROUP_ID", "inventory-group")
	instanceID := getEnv("HOSTNAME", "inventory-1")

	dbConfig := db.DefaultConfig()
	dbConfig.Host = getEnv("DB_HOST", "localhost")
	dbConfig.User = getEnv("DB_USER", "postgres")
	dbConfig.Password = getEnv("DB_PASSWORD", "postgres")
	dbConfig.DBName = getEnv("DB_NAME", "inventory_db")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	service, err := NewInventoryService(brokers, groupID, instanceID, []string{"orders"}, pool)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigterm
		cancel()
	}()

	if err := service.ProcessOrders(ctx); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
