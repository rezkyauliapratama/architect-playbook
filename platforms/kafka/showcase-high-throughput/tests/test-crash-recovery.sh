#!/bin/bash

echo "🧪 TEST 5: CONSUMER CRASH RECOVERY (IMPROVED)"
echo "=============================================="
echo "Scenario: Crash consumer while processing messages"
echo "Expected problem: Auto-commit causes duplicates or data loss"
echo ""

# Reset
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    UPDATE products SET reserved_quantity = 0;
    TRUNCATE processed_orders, inventory_logs;
" > /dev/null 2>&1

echo "Phase 1: Sending 100 orders rapidly..."

# Send 100 orders fast
for i in {1..100}; do
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"usr_crash_${i}\",
      \"product_id\": \"prd_laptop_001\",
      \"quantity\": 1
    }" > /dev/null &
  
  # Batch control
  if [ $(( i % 20 )) -eq 0 ]; then
    wait
  fi
done

wait
echo "✓ All 100 orders sent"

# Wait briefly for some processing
echo ""
echo "Phase 2: Letting consumer process some messages (2 seconds)..."
sleep 2

# Check status BEFORE crash
BEFORE_CRASH=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo "✓ Consumer processed: $BEFORE_CRASH orders"

# CRASH the consumer
echo ""
echo "Phase 3: 💥 CRASHING CONSUMER NOW..."
docker kill -s SIGKILL showcase-high-throughput-inventory-service-1 > /dev/null 2>&1

echo "✓ Consumer killed (SIGKILL - hard crash)"
echo ""
echo "Phase 4: Waiting 3 seconds before restart..."
sleep 3

# Check Kafka lag
echo "Checking Kafka consumer lag..."
docker exec redpanda rpk group describe inventory-consumer-group 2>/dev/null || echo "Consumer group offline"

# Restart consumer
echo ""
echo "Phase 5: Restarting consumer..."
docker restart showcase-high-throughput-inventory-service-1 > /dev/null 2>&1

echo "✓ Consumer restarted"
echo ""
echo "Phase 6: Waiting for recovery (10 seconds)..."
sleep 10

# Check AFTER recovery
AFTER_RECOVERY=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo ""
echo "Results:"
echo "========="
echo "Before crash:       $BEFORE_CRASH orders"
echo "After recovery:     $AFTER_RECOVERY orders"
echo "Total sent:         100 orders"
echo ""

# Check for duplicates
echo "Checking for duplicates..."
DUPLICATE_ANALYSIS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    SELECT 
        user_id,
        COUNT(*) as times_processed
    FROM processed_orders
    GROUP BY user_id
    HAVING COUNT(*) > 1
    ORDER BY times_processed DESC
    LIMIT 10;
")

DUPLICATE_COUNT=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(DISTINCT user_id) 
    FROM (
        SELECT user_id
        FROM processed_orders
        GROUP BY user_id
        HAVING COUNT(*) > 1
    ) sub;
" | tr -d ' ')

echo ""
if [ "$DUPLICATE_COUNT" -gt 0 ]; then
    echo "❌ DUPLICATE PROCESSING DETECTED!"
    echo "   $DUPLICATE_COUNT orders processed multiple times"
    echo ""
    echo "Duplicate details:"
    echo "$DUPLICATE_ANALYSIS"
    echo ""
    echo "Root cause: Auto-commit commits offsets BEFORE processing"
    echo "           Crash causes reprocessing of committed but failed messages"
else
    echo "✅ No duplicates detected"
fi

# Check for data loss
LOST=$((100 - AFTER_RECOVERY + DUPLICATE_COUNT))
if [ "$LOST" -gt 0 ]; then
    echo ""
    echo "❌ DATA LOSS DETECTED!"
    echo "   Lost: $LOST orders"
fi

# Show consumer lag
echo ""
echo "Final consumer lag:"
docker exec redpanda rpk group describe inventory-consumer-group 2>/dev/null | grep -A 10 "PARTITION"

echo ""
echo "Recovery analysis:"
echo "------------------"
echo "Recovery time:      ~13 seconds (3s wait + 10s processing)"
echo "Messages lost:      $LOST"
echo "Messages duplicate: $DUPLICATE_COUNT"
echo "Success rate:       $(( (AFTER_RECOVERY - DUPLICATE_COUNT) * 100 / 100 ))%"
