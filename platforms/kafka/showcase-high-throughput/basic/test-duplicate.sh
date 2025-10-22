#!/bin/bash

echo "🧪 TEST 2: DUPLICATE PROCESSING SIMULATION"
echo "==========================================="
echo "Scenario: Send same order 3 times (simulate network retry)"
echo "Expected (WRONG): Order processed multiple times"
echo ""

# Reset
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    UPDATE products SET reserved_quantity = 0;
    TRUNCATE processed_orders, inventory_logs;
"

# Send same order 3 times
ORDER_DATA='{
  "user_id": "usr_duplicate_test",
  "product_id": "prd_mouse_002",
  "quantity": 10
}'

echo "Sending identical order 3 times (1 second apart)..."
for i in {1..3}; do
  echo "Attempt $i..."
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "$ORDER_DATA" | jq -r '.order_id'
  sleep 1
done

sleep 2

echo ""
echo "Results:"
echo "--------"

# Count unique users
UNIQUE_ORDERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) 
    FROM processed_orders 
    WHERE user_id = 'usr_duplicate_test';
" | tr -d ' ')

TOTAL_RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT SUM(quantity) 
    FROM processed_orders 
    WHERE user_id = 'usr_duplicate_test';
" | tr -d ' ')

echo "Orders processed: $UNIQUE_ORDERS"
echo "Total reserved:   $TOTAL_RESERVED units"
echo "Expected:         1 order × 10 units = 10 units"
echo ""

if [ "$UNIQUE_ORDERS" -gt 1 ]; then
    echo "❌ DUPLICATE PROCESSING DETECTED!"
    echo "   Same order processed $UNIQUE_ORDERS times"
    echo "   Overselling: $(( TOTAL_RESERVED - 10 )) units"
else
    echo "✅ No duplicates (unlikely in baseline)"
fi

echo ""
echo "View all orders:"
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    SELECT order_id, user_id, quantity, processed_at 
    FROM processed_orders 
    WHERE user_id = 'usr_duplicate_test';
"
