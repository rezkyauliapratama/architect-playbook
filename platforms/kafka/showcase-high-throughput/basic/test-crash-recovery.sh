#!/bin/bash

echo "🧪 TEST 5: CONSUMER CRASH RECOVERY"
echo "==================================="
echo "Scenario: Send orders, crash consumer mid-processing, check duplicates"
echo ""

# Reset
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    UPDATE products SET reserved_quantity = 0;
    TRUNCATE processed_orders, inventory_logs;
"

# Send 50 orders
echo "Phase 1: Sending 50 orders..."
for i in {1..50}; do
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"usr_crash_$i\",
      \"product_id\": \"prd_laptop_001\",
      \"quantity\": 1
    }" > /dev/null &
done
wait

echo "Waiting for processing to start (2s)..."
sleep 2

# Check how many processed before crash
BEFORE_CRASH=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo "Processed before crash: $BEFORE_CRASH orders"

# Crash consumer
echo ""
echo "Phase 2: Simulating consumer crash..."
docker restart inventory-service

echo "Waiting for recovery (5s)..."
sleep 5

# Check after recovery
AFTER_RECOVERY=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo ""
echo "Results:"
echo "--------"
echo "Before crash:  $BEFORE_CRASH orders"
echo "After recovery: $AFTER_RECOVERY orders"
echo "Total sent:     50 orders"

# Check for duplicates
DUPLICATES=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FILTER (WHERE cnt > 1) as duplicates
    FROM (
        SELECT order_id, COUNT(*) as cnt
        FROM processed_orders
        GROUP BY order_id
    ) sub;
" | tr -d ' ')

echo "Duplicates:     $DUPLICATES orders"

if [ "$DUPLICATES" -gt 0 ]; then
    echo ""
    echo "❌ DUPLICATE PROCESSING AFTER CRASH!"
    echo "   Auto-commit caused $(( DUPLICATES * 100 / 50 ))% duplicate rate"
fi

# Check for data loss
LOST=$((50 - (AFTER_RECOVERY - DUPLICATES)))
if [ "$LOST" -gt 0 ]; then
    echo "❌ DATA LOSS: $LOST orders lost"
fi
