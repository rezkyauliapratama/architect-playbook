#!/bin/bash

echo "🧪 TEST 4: MESSAGE ORDERING TEST"
echo "================================="
echo "Scenario: Send 5 events for same order (create→update→update→complete→cancel)"
echo "Expected order: 1→2→3→4→5"
echo ""

ORDER_BASE="ord_sequence_$(date +%s)"

# Send events with sequence
echo "Sending 5 events in sequence..."
for seq in 1 2 3 4 5; do
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"usr_sequence\",
      \"product_id\": \"prd_monitor_004\",
      \"quantity\": ${seq}
    }" | jq -r '.order_id'
  sleep 0.1
done

echo ""
echo "Waiting for processing (3s)..."
sleep 3

echo ""
echo "Processing order:"
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    SELECT 
        order_id,
        quantity as sequence_num,
        processed_at 
    FROM processed_orders 
    WHERE user_id = 'usr_sequence' 
    ORDER BY processed_at;
"

echo ""
echo "Check if sequence matches send order..."
SEQUENCES=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT array_agg(quantity ORDER BY processed_at) 
    FROM processed_orders 
    WHERE user_id = 'usr_sequence';
" | tr -d ' {}')

echo "Expected sequence: 1,2,3,4,5"
echo "Actual sequence:   $SEQUENCES"

if [ "$SEQUENCES" != "1,2,3,4,5" ]; then
    echo ""
    echo "❌ OUT-OF-ORDER PROCESSING DETECTED!"
    echo "   Messages processed in wrong sequence"
fi
