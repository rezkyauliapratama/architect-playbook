#!/bin/bash

echo "🧪 TEST 1: RACE CONDITION SIMULATION (Baseline)"
echo "================================================"
echo "Scenario: 10 concurrent requests for SAME order"
echo "Expected (WRONG): Some orders processed multiple times"
echo ""

# Reset inventory
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    UPDATE products SET reserved_quantity = 0 WHERE product_id = 'prd_laptop_001';
    TRUNCATE processed_orders, inventory_logs;
"

echo "Initial stock: 100 laptops"
echo ""

# Create same order ID (simulate retry/duplicate)
ORDER_ID="test_race_$(date +%s)"

echo "Sending 10 concurrent requests with SAME order details..."
for i in {1..10}; do
  (
    curl -s -X POST http://localhost:8081/orders \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"usr_race\",
        \"product_id\": \"prd_laptop_001\",
        \"quantity\": 5
      }" > /dev/null
  ) &
done

# Wait for all requests
wait
sleep 2

echo ""
echo "Results:"
echo "--------"

# Check processed count
PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

# Check reserved quantity
RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT reserved_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

echo "Orders processed: $PROCESSED"
echo "Total reserved:   $RESERVED units"
echo "Expected:         10 orders × 5 units = 50 units"
echo ""

if [ "$PROCESSED" -gt 10 ] || [ "$RESERVED" -gt 50 ]; then
    echo "❌ RACE CONDITION DETECTED!"
    echo "   System processed duplicate orders"
    echo "   Race condition rate: $(( (RESERVED - 50) * 100 / 50 ))%"
else
    echo "✅ No race condition (but this is unlikely in baseline)"
fi

echo ""
echo "View details:"
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    SELECT order_id, quantity, processed_at 
    FROM processed_orders 
    WHERE product_id = 'prd_laptop_001' 
    ORDER BY processed_at;
"
