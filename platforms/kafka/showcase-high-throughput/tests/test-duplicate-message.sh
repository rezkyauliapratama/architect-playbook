#!/bin/bash

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  TEST 1: DUPLICATE MESSAGE PROCESSING (BASELINE)     ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "Scenario: Producer retries cause duplicate messages in Kafka"
echo "Baseline:  No idempotence → Same message sent multiple times"
echo "Impact:    Consumer processes same order multiple times"
echo ""

# Reset database
echo "📋 Resetting database..."
docker exec inventory-postgres psql -U postgres -d inventory_db -q <<EOF
UPDATE products SET reserved_quantity = 0, stock_quantity = 1000 WHERE product_id = 'prd_laptop_001';
DELETE FROM processed_orders;
DELETE FROM inventory_logs;
EOF

INITIAL_STOCK=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT stock_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

echo -e "${GREEN}✓ Initial stock: ${INITIAL_STOCK} units${NC}"
echo ""

echo "🚀 Test Configuration:"
echo "   Product:      prd_laptop_001"
echo "   Orders:       20 unique users"
echo "   Duplicates:   Each order sent 3 times (simulate retry)"
echo "   Qty/order:    10 units"
echo "   Expected:     200 units reserved (20 × 10)"
echo ""

# Test parameters
PRODUCT_ID="prd_laptop_001"
QUANTITY=10
NUM_USERS=20
RETRIES=3  # Send each order 3 times

echo "⏳ Sending orders with simulated retries..."
echo ""

# Send orders - each order sent RETRIES times to simulate network retry
for user in $(seq 1 $NUM_USERS); do
  USER_ID="usr_duplicate_${user}"
  
  echo "  User ${user}: Sending order ${RETRIES}x (retry simulation)..."
  
  for retry in $(seq 1 $RETRIES); do
    # Send same order multiple times (simulate producer retry without idempotence)
    curl -s -X POST http://localhost:8081/orders \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"${USER_ID}\",
        \"product_id\": \"${PRODUCT_ID}\",
        \"quantity\": ${QUANTITY}
      }" > /dev/null &
    
    # Small delay between retries to ensure they go to different Kafka partitions
    sleep 0.05
  done
  
  # Wait every 5 users
  if [ $((user % 5)) -eq 0 ]; then
    wait
  fi
done

wait
echo ""
echo -e "${GREEN}✓ All ${NUM_USERS} orders sent (${RETRIES}x each = $((NUM_USERS * RETRIES)) total messages)${NC}"
echo ""

echo "⏳ Waiting for consumer processing (15 seconds)..."
sleep 15

# Collect results
echo ""
echo "═══════════════════════════════════════════════════════"
echo "                   RESULTS ANALYSIS                     "
echo "═══════════════════════════════════════════════════════"
echo ""

TOTAL_MESSAGES=$((NUM_USERS * RETRIES))
EXPECTED_RESERVED=$((NUM_USERS * QUANTITY))

TOTAL_PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

UNIQUE_USERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(DISTINCT user_id) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT reserved_quantity FROM products WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

# Calculate issues
DUPLICATES=$((TOTAL_PROCESSED - UNIQUE_USERS))
DUPLICATE_RATE=0
if [ "$UNIQUE_USERS" -gt 0 ]; then
    DUPLICATE_RATE=$(( (DUPLICATES * 100) / UNIQUE_USERS ))
fi

OVER_RESERVED=$((RESERVED - EXPECTED_RESERVED))
OVER_PERCENT=0
if [ "$EXPECTED_RESERVED" -gt 0 ]; then
    OVER_PERCENT=$(( (OVER_RESERVED * 100) / EXPECTED_RESERVED ))
fi

echo "📊 Message Flow:"
echo "   Messages sent to Kafka:  $TOTAL_MESSAGES (${NUM_USERS} orders × ${RETRIES} retries)"
echo "   Unique users:            $UNIQUE_USERS"
echo "   Total DB records:        $TOTAL_PROCESSED"
echo "   Duplicate records:       $DUPLICATES"
echo ""

echo "📦 Inventory Impact:"
echo "   Expected reservation:    ${EXPECTED_RESERVED} units"
echo "   Actual reservation:      ${RESERVED} units"
echo "   Over-reserved:           ${OVER_RESERVED} units"
echo ""

# Analysis
if [ "$DUPLICATES" -gt 0 ] || [ "$OVER_RESERVED" -gt 0 ]; then
    echo -e "${RED}❌ DUPLICATE PROCESSING DETECTED!${NC}"
    echo ""
    
    if [ "$DUPLICATES" -gt 0 ]; then
        echo -e "${RED}   Database Duplicates:${NC}"
        echo "   - Duplicate records:  ${DUPLICATES}"
        echo "   - Duplicate rate:     ${DUPLICATE_RATE}%"
        echo ""
    fi
    
    if [ "$OVER_RESERVED" -gt 0 ]; then
        echo -e "${RED}   Inventory Over-Reservation:${NC}"
        echo "   - Over-reserved:      ${OVER_RESERVED} units"
        echo "   - Over-reservation:   ${OVER_PERCENT}%"
        echo ""
    fi
    
    echo "🔍 Root Cause Analysis:"
    echo "   1. Producer sends order without idempotence"
    echo "   2. Network/timeout causes retry"
    echo "   3. Same order appears 3× in Kafka (different messages)"
    echo "   4. Consumer processes all messages"
    echo "   5. No idempotency check → Each message reserves inventory"
    echo "   6. Result: ${RESERVED} units reserved instead of ${EXPECTED_RESERVED}"
    echo ""
    
    # Show users with multiple processing
    echo "📋 Users Processed Multiple Times (Top 10):"
    docker exec inventory-postgres psql -U postgres -d inventory_db <<EOF
SELECT 
    user_id,
    COUNT(*) as times_processed,
    SUM(quantity) as total_reserved,
    MIN(processed_at) as first_processed,
    MAX(processed_at) as last_processed,
    EXTRACT(MILLISECONDS FROM (MAX(processed_at) - MIN(processed_at))) as span_ms
FROM processed_orders
WHERE product_id = '${PRODUCT_ID}'
GROUP BY user_id
HAVING COUNT(*) > 1
ORDER BY COUNT(*) DESC
LIMIT 10;
EOF
    
    echo ""
    echo "💡 Impact:"
    echo "   - Inventory over-reserved by ${OVER_RESERVED} units"
    echo "   - ${DUPLICATES} duplicate database records"
    echo "   - Data inconsistency: DB shows ${TOTAL_PROCESSED} orders, actually ${UNIQUE_USERS} unique"
    echo ""
    echo -e "${RED}Result: BASELINE SYSTEM FAILED${NC}"
    echo -e "${RED}        Duplicate processing rate: ${DUPLICATE_RATE}%${NC}"
    
elif [ "$TOTAL_PROCESSED" -eq "$NUM_USERS" ] && [ "$RESERVED" -eq "$EXPECTED_RESERVED" ]; then
    echo -e "${GREEN}✅ NO DUPLICATES DETECTED${NC}"
    echo "   System correctly processed each order once"
    echo ""
    echo -e "${YELLOW}Note: This means idempotency is somehow working${NC}"
    echo "      Check if Pattern 1 already applied to baseline"
    
else
    echo -e "${YELLOW}⚠️  INCONCLUSIVE RESULTS${NC}"
    echo "   Processed: $TOTAL_PROCESSED, Expected: $NUM_USERS"
    echo "   Reserved: $RESERVED, Expected: $EXPECTED_RESERVED"
fi

# Show inventory state
echo ""
echo "📦 Final Inventory State:"
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
SELECT 
    product_id,
    stock_quantity as stock,
    reserved_quantity as reserved,
    (stock_quantity - reserved_quantity) as available,
    CASE 
        WHEN reserved_quantity > stock_quantity THEN '❌ OVERSOLD'
        WHEN reserved_quantity = 200 THEN '❌ DUPLICATE'
        ELSE '✓ OK'
    END as status
FROM products 
WHERE product_id = '${PRODUCT_ID}';
"

echo ""
echo "═══════════════════════════════════════════════════════"
echo "                    TEST COMPLETE                       "
echo "═══════════════════════════════════════════════════════"
