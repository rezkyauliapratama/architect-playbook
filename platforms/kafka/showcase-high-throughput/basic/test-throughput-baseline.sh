#!/bin/bash

echo "🧪 TEST 3: THROUGHPUT MEASUREMENT (Baseline)"
echo "============================================="
echo "Scenario: Send 1,000 orders as fast as possible"
echo ""

# Reset
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    UPDATE products SET reserved_quantity = 0;
    TRUNCATE processed_orders, inventory_logs;
"

START_TIME=$(date +%s)

echo "Sending 1,000 orders..."
for i in {1..1000}; do
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"usr_throughput_$i\",
      \"product_id\": \"prd_keyboard_003\",
      \"quantity\": 1
    }" > /dev/null &
  
  # Batch every 50 to prevent overwhelming
  if [ $(( i % 50 )) -eq 0 ]; then
    wait
    echo "  Sent $i orders..."
  fi
done

wait
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "All requests sent in $DURATION seconds"
echo "Waiting for processing (10s)..."
sleep 10

# Count processed
PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo ""
echo "Results:"
echo "--------"
echo "Sent:              1,000 orders"
echo "Time taken:        $DURATION seconds"
echo "Processed:         $PROCESSED orders"
echo "Success rate:      $(( PROCESSED * 100 / 1000 ))%"
echo "Throughput:        $(( 1000 / DURATION )) orders/sec"
echo "Avg latency:       $(( DURATION * 1000 / 1000 ))ms per order"

if [ "$PROCESSED" -lt 1000 ]; then
    LOST=$((1000 - PROCESSED))
    echo ""
    echo "❌ MESSAGE LOSS DETECTED!"
    echo "   Lost: $LOST orders ($(( LOST * 100 / 1000 ))%)"
fi

# Check for duplicates
DUPLICATES=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) - COUNT(DISTINCT order_id) 
    FROM processed_orders;
" | tr -d ' ')

if [ "$DUPLICATES" -gt 0 ]; then
    echo "❌ DUPLICATES: $DUPLICATES orders processed twice"
fi
