#!/bin/bash

echo "🧪 TEST 4: OUT-OF-ORDER TEST (HEAVY LOAD)"
echo "=========================================="
echo "Scenario: Multiple producers sending to same topic simultaneously"
echo ""

# Reset
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    UPDATE products SET reserved_quantity = 0;
    TRUNCATE processed_orders, inventory_logs;
" > /dev/null 2>&1

echo "Sending 200 orders from 4 parallel producers..."

# Launch 4 producers in parallel
for producer in {1..4}; do
  (
    for seq in $(seq $producer 4 200); do
      curl -s -X POST http://localhost:8081/orders \
        -H "Content-Type: application/json" \
        -d "{
          \"user_id\": \"usr_parallel_test\",
          \"product_id\": \"prd_laptop_001\",
          \"quantity\": ${seq}
        }" > /dev/null
    done
  ) &
done

wait
echo "✓ All 200 orders sent"
sleep 5

# Analyze
OUT_OF_ORDER=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    WITH numbered AS (
        SELECT 
            quantity as sequence_sent,
            ROW_NUMBER() OVER (ORDER BY processed_at) as position
        FROM processed_orders 
        WHERE user_id = 'usr_parallel_test'
    )
    SELECT COUNT(*) 
    FROM numbered 
    WHERE sequence_sent != position;
" | tr -d ' ')

TOTAL=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) 
    FROM processed_orders 
    WHERE user_id = 'usr_parallel_test';
" | tr -d ' ')

echo ""
echo "Results:"
echo "--------"
echo "Total processed:    $TOTAL"
echo "Out-of-order:       $OUT_OF_ORDER"
echo "Out-of-order rate:  $(( OUT_OF_ORDER * 100 / TOTAL ))%"

if [ "$OUT_OF_ORDER" -gt 20 ]; then
    echo ""
    echo "❌ SIGNIFICANT OUT-OF-ORDER PROCESSING!"
    echo "   $(( OUT_OF_ORDER * 100 / TOTAL ))% of messages scrambled"
fi
