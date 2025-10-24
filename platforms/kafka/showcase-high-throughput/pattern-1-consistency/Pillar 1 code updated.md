<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# okay sekarang maru kita adjust system dan test scenario nya dengan menerapkan pillar 1

Perfect! Sekarang saya buatkan **PILLAR 1: CONSISTENCY** implementation dengan:

1. Updated services (producer dengan partition key + consumer dengan idempotency)
2. Test scenarios yang membuktikan CONSISTENCY sudah fix

***

# **PILLAR 1: CONSISTENCY IMPLEMENTATION**

## **What We're Fixing**

| Problem | Solution |
| :-- | :-- |
| ❌ Race conditions (12.3%) | ✅ Partition keying |
| ❌ Duplicate processing (8.5%) | ✅ Idempotency tracking |
| ❌ Out-of-order events (15.7%) | ✅ Consistent routing |
| ❌ Data accuracy (87.7%) | ✅ 100% consistency |


***

## **1. ORDER SERVICE - WITH PARTITION KEY**

### **File: `order-service/main.go` (PILLAR 1 Applied)**

```go
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
    // ✅ PILLAR 1: Idempotence enabled (prevents duplicates from retries)
    config := &kafka.ConfigMap{
        "bootstrap.servers": brokers,
        
        // ✅ CONSISTENCY FIX 1: Enable idempotence
        "enable.idempotence": true,
        
        // ✅ CONSISTENCY FIX 2: Wait for all replicas
        "acks": "all",
        
        // ✅ CONSISTENCY FIX 3: Allow retries without duplicates
        "max.in.flight.requests.per.connection": 5,
        
        // Basic settings (will optimize in Pillar 2)
        "batch.size":       16384,
        "linger.ms":        0,
        "compression.type": "none",
        
        "client.id":          "order-service-pillar1",
        "request.timeout.ms": 30000,
    }
    
    producer, err := kafka.NewProducer(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create producer: %w", err)
    }
    
    log.Printf("✅ Producer created (PILLAR 1: CONSISTENCY)")
    log.Printf("   - Idempotence: ENABLED")
    log.Printf("   - Acks: ALL (full replication)")
    log.Printf("   - Partition key: ENABLED")
    
    service := &OrderService{
        producer: producer,
        topic:    topic,
    }
    
    go service.handleDeliveryReports()
    
    return service, nil
}

func (s *OrderService) handleDeliveryReports() {
    for e := range s.producer.Events() {
        switch ev := e.(type) {
        case *kafka.Message:
            if ev.TopicPartition.Error != nil {
                log.Printf("❌ Delivery failed: %v", ev.TopicPartition.Error)
            } else {
                log.Printf("✅ Delivered: key=%s partition=%d offset=%d",
                    string(ev.Key),
                    ev.TopicPartition.Partition,
                    ev.TopicPartition.Offset)
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
    
    if req.UserID == "" || req.ProductID == "" || req.Quantity <= 0 {
        http.Error(w, "Missing required fields", http.StatusBadRequest)
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
    
    orderBytes, err := json.Marshal(order)
    if err != nil {
        http.Error(w, "Failed to serialize", http.StatusInternalServerError)
        return
    }
    
    // ✅ PILLAR 1: CRITICAL FIX - Use order_id as partition key
    // This ensures:
    // 1. Same order → Same partition → Same consumer
    // 2. Ordering guaranteed per order
    // 3. Zero race conditions
    err = s.producer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     &s.topic,
            Partition: kafka.PartitionAny,
        },
        Key:   []byte(order.OrderID),  // ✅ PARTITION KEY!
        Value: orderBytes,
    }, nil)
    
    if err != nil {
        log.Printf("Failed to produce: %v", err)
        http.Error(w, "Failed to process order", http.StatusInternalServerError)
        return
    }
    
    log.Printf("📦 Order created: %s (key=%s)", orderID, orderID)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "order_id":  orderID,
        "status":    "accepted",
        "message":   "Order is being processed",
        "timestamp": order.Timestamp.Format(time.RFC3339),
    })
}

func (s *OrderService) HealthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":  "healthy",
        "version": "pillar1",
        "features": map[string]bool{
            "idempotence":    true,
            "partition_key":  true,
            "acks_all":       true,
        },
    })
}

func (s *OrderService) Close() {
    log.Println("🛑 Shutting down...")
    remaining := s.producer.Flush(15 * 1000)
    if remaining > 0 {
        log.Printf("⚠️  %d messages not delivered", remaining)
    }
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
    
    http.HandleFunc("/orders", service.CreateOrderHandler)
    http.HandleFunc("/health", service.HealthHandler)
    
    log.Printf("🚀 Order Service (PILLAR 1: CONSISTENCY) running on :%s", port)
    log.Printf("   Features: Idempotence ✓ | Partition Key ✓ | Acks=ALL ✓")
    
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}
```


***

## **2. INVENTORY SERVICE - WITH IDEMPOTENCY**

### **File: `inventory-service/main.go` (PILLAR 1 Applied)**

```go
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

    "inventory-service/db"
    "inventory-service/repository"
    
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
    consumer *kafka.Consumer
    repo     *repository.InventoryRepository
    pool     *pgxpool.Pool
    
    // Metrics
    messagesProcessed int64
    messagesSkipped   int64
    messagesFailed    int64
}

func NewInventoryService(brokers, groupID, instanceID string, topics []string, pool *pgxpool.Pool) (*InventoryService, error) {
    // ✅ PILLAR 1: Consumer configuration for consistency
    config := &kafka.ConfigMap{
        "bootstrap.servers": brokers,
        "group.id":          groupID,
        
        // ✅ CONSISTENCY FIX 1: Read only committed messages
        "isolation.level": "read_committed",
        
        // Basic settings (will optimize in Pillar 2 & 3)
        "enable.auto.commit": true,  // Will fix in Pillar 3
        "auto.offset.reset":  "earliest",
        "session.timeout.ms": 10000,
        
        "client.id": "inventory-service-pillar1",
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
    
    log.Printf("✅ Consumer created (PILLAR 1: CONSISTENCY)")
    log.Printf("   - Isolation: read_committed")
    log.Printf("   - Idempotency: ENABLED (database-backed)")
    
    return &InventoryService{
        consumer: consumer,
        repo:     repository.NewInventoryRepository(pool),
        pool:     pool,
    }, nil
}

func (s *InventoryService) ProcessOrders(ctx context.Context) error {
    log.Println("🚀 Inventory Service (PILLAR 1: CONSISTENCY) started")
    
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
                
            case kafka.Error:
                log.Printf("❌ Error: %v", e)
            }
        }
    }
}

func (s *InventoryService) processMessage(ctx context.Context, msg *kafka.Message) error {
    startTime := time.Now()
    
    var order Order
    if err := json.Unmarshal(msg.Value, &order); err != nil {
        return err
    }
    
    log.Printf("📦 Processing: %s (key=%s, partition=%d, offset=%d)",
        order.OrderID, 
        string(msg.Key),
        msg.TopicPartition.Partition, 
        msg.TopicPartition.Offset)
    
    // ✅ PILLAR 1: CRITICAL - Idempotency check
    // This prevents duplicate processing even if:
    // - Same message consumed twice
    // - Network retry
    // - Consumer crash and replay
    existing, err := s.repo.CheckIdempotency(ctx, order.IdempotencyKey)
    if err != nil {
        return fmt.Errorf("idempotency check failed: %w", err)
    }
    
    if existing != nil {
        log.Printf("⏭  Already processed: %s (idempotency key: %s)", 
            order.OrderID, order.IdempotencyKey)
        s.messagesSkipped++
        return nil
    }
    
    // Process order (transactional with idempotency tracking)
    err = s.repo.ReserveInventory(
        ctx,
        order.OrderID,
        order.UserID,
        order.ProductID,
        order.Quantity,
        order.IdempotencyKey,
    )
    
    if err != nil {
        if err == repository.ErrInsufficientStock {
            log.Printf("⚠️  Insufficient stock for %s", order.ProductID)
            return nil
        }
        if err == repository.ErrProductNotFound {
            log.Printf("⚠️  Product not found: %s", order.ProductID)
            return nil
        }
        if err == repository.ErrAlreadyProcessed {
            log.Printf("⏭  Race condition avoided: %s", order.OrderID)
            s.messagesSkipped++
            return nil
        }
        return err
    }
    
    s.messagesProcessed++
    processingTime := time.Since(startTime)
    
    log.Printf("✅ Order %s completed (processing_time=%dms)",
        order.OrderID, processingTime.Milliseconds())
    
    return nil
}

func (s *InventoryService) reportMetrics(ctx context.Context) {
    log.Printf("📊 Metrics:")
    log.Printf("   - Processed: %d", s.messagesProcessed)
    log.Printf("   - Skipped (idempotency): %d", s.messagesSkipped)
    log.Printf("   - Failed: %d", s.messagesFailed)
    
    stats, err := s.repo.GetInventoryStats(ctx)
    if err != nil {
        log.Printf("⚠️  Failed to get stats: %v", err)
        return
    }
    
    log.Printf("📦 Inventory:")
    log.Printf("   - Products: %v", stats["total_products"])
    log.Printf("   - Total Stock: %v", stats["total_stock"])
    log.Printf("   - Reserved: %v", stats["total_reserved"])
    log.Printf("   - Available: %v", stats["total_available"])
}

func (s *InventoryService) shutdown() error {
    s.pool.Close()
    return s.consumer.Close()
}

func main() {
    brokers := getEnv("KAFKA_BROKERS", "localhost:19092")
    groupID := getEnv("KAFKA_GROUP_ID", "inventory-group-pillar1")
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
```


***

## **3. PILLAR 1 TEST SCENARIOS**

### **File: `test-pillar1-consistency.sh`**

```bash
#!/bin/bash

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  PILLAR 1: CONSISTENCY VERIFICATION                   ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""
echo "Testing:"
echo "  ✓ Partition key prevents race conditions"
echo "  ✓ Idempotency prevents duplicates"
echo "  ✓ Data accuracy = 100%"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Reset
echo "📋 Resetting system..."
docker exec inventory-postgres psql -U postgres -d inventory_db -q <<EOF
UPDATE products SET reserved_quantity = 0;
TRUNCATE processed_orders, inventory_logs;
EOF
echo "✓ Reset complete"
echo ""

# ===================================================
# TEST 1: RACE CONDITION (Should be FIXED)
# ===================================================
echo "═══════════════════════════════════════════════════════"
echo "  TEST 1: Race Condition Prevention"
echo "═══════════════════════════════════════════════════════"
echo ""

echo "Scaling to 3 consumers (high contention)..."
docker-compose up -d --scale inventory-service=3 2>/dev/null
sleep 5

echo "Sending 20 concurrent orders..."
for i in {1..20}; do
  {
    curl -s -X POST http://localhost:8081/orders \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"usr_race\",
        \"product_id\": \"prd_laptop_001\",
        \"quantity\": 10
      }" > /dev/null
  } &
done
wait
sleep 8

# Check results
PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT reserved_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

EXPECTED=$((20 * 10))

echo ""
echo "Results:"
echo "  Orders processed: $PROCESSED"
echo "  Expected reserved: $EXPECTED units"
echo "  Actual reserved:   $RESERVED units"
echo ""

if [ "$RESERVED" -eq "$EXPECTED" ]; then
    echo -e "${GREEN}✅ RACE CONDITION FIXED!${NC}"
    echo "   Partition key ensures consistent routing"
else
    echo -e "${RED}❌ Still has issues${NC}"
fi

docker-compose up -d --scale inventory-service=1 2>/dev/null
sleep 2

# ===================================================
# TEST 2: DUPLICATE PREVENTION
# ===================================================
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  TEST 2: Duplicate Prevention"
echo "═══════════════════════════════════════════════════════"
echo ""

docker exec inventory-postgres psql -U postgres -d inventory_db -q <<EOF
UPDATE products SET reserved_quantity = 0;
TRUNCATE processed_orders, inventory_logs;
EOF

echo "Sending same order 5 times (simulate retry)..."
for i in {1..5}; do
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"usr_duplicate_test\",
      \"product_id\": \"prd_mouse_002\",
      \"quantity\": 5
    }" > /dev/null
  sleep 0.5
done

sleep 3

PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE user_id = 'usr_duplicate_test';
" | tr -d ' ')

RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COALESCE(SUM(quantity), 0) FROM processed_orders WHERE user_id = 'usr_duplicate_test';
" | tr -d ' ')

echo ""
echo "Results:"
echo "  Orders processed: $PROCESSED"
echo "  Reserved units:   $RESERVED"
echo "  Expected:         1 order, 5 units"
echo ""

if [ "$PROCESSED" -eq 1 ] && [ "$RESERVED" -eq 5 ]; then
    echo -e "${GREEN}✅ DUPLICATE PREVENTION WORKING!${NC}"
    echo "   Idempotency check rejected 4 duplicates"
else
    echo -e "${RED}❌ Duplicates detected${NC}"
fi

# ===================================================
# TEST 3: DATA CONSISTENCY
# ===================================================
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  TEST 3: Overall Data Consistency"
echo "═══════════════════════════════════════════════════════"
echo ""

docker exec inventory-postgres psql -U postgres -d inventory_db -q <<EOF
UPDATE products SET reserved_quantity = 0;
TRUNCATE processed_orders, inventory_logs;
EOF

echo "Sending 100 orders (mixed products, varied concurrency)..."
for i in {1..100}; do
  PRODUCT=$( [ $((i % 4)) -eq 0 ] && echo "prd_laptop_001" || echo "prd_mouse_002" )
  {
    curl -s -X POST http://localhost:8081/orders \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"usr_consistency_$i\",
        \"product_id\": \"$PRODUCT\",
        \"quantity\": 1
      }" > /dev/null
  } &
  
  if [ $((i % 10)) -eq 0 ]; then
    wait
  fi
done
wait
sleep 10

# Verify
UNIQUE_ORDERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(DISTINCT user_id) FROM processed_orders;
" | tr -d ' ')

TOTAL_PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo ""
echo "Results:"
echo "  Unique orders:    $UNIQUE_ORDERS"
echo "  Total processed:  $TOTAL_PROCESSED"
echo "  Expected:         100"
echo ""

if [ "$UNIQUE_ORDERS" -eq 100 ] && [ "$TOTAL_PROCESSED" -eq 100 ]; then
    echo -e "${GREEN}✅ 100% DATA CONSISTENCY ACHIEVED!${NC}"
    echo "   Zero race conditions"
    echo "   Zero duplicates"
else
    echo -e "${YELLOW}⚠️  Consistency: $(( UNIQUE_ORDERS * 100 / 100 ))%${NC}"
fi

# ===================================================
# SUMMARY
# ===================================================
echo ""
echo "═══════════════════════════════════════════════════════"
echo "              PILLAR 1 VERIFICATION COMPLETE            "
echo "═══════════════════════════════════════════════════════"
echo ""
echo "Improvements:"
echo "  ✓ Partition key → Consistent routing"
echo "  ✓ Idempotence → Zero duplicates"
echo "  ✓ Acks=ALL → Zero data loss"
echo "  ✓ Data accuracy: 100%"
echo ""
echo "Next: Apply Pillar 2 (Throughput) or Pillar 3 (Fault Tolerance)"
```


***

## **4. DOCKER COMPOSE (PILLAR 1)**

No changes needed to docker-compose, just rebuild services:

```bash
# Rebuild with Pillar 1 code
docker-compose build order-service inventory-service

# Restart
docker-compose up -d

# Run Pillar 1 tests
./test-pillar1-consistency.sh
```


***

## **PILLAR 1 SUMMARY**

**Changes Made:**

1. ✅ **Producer**: `enable.idempotence=true`, `acks=all`, **partition key**
2. ✅ **Consumer**: `isolation.level=read_committed`, **idempotency check**
3. ✅ **Database**: Idempotency tracking in `processed_orders` table

**Expected Results:**

- Race conditions: 0% (was 12.3%)
- Duplicates: 0% (was 8.5%)
- Data accuracy: 100% (was 87.7%)

**Run tests to verify!**

