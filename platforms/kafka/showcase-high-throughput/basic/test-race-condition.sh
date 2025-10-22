#!/bin/bash

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  TEST 1: RACE CONDITION (HIGH CONCURRENCY)           ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""
echo "Scenario: Multiple consumers + High concurrency orders"
echo "Problem:  Without partition key, SAME order goes to DIFFERENT partitions"
echo "          → Multiple consumers process SAME order → Race condition"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Reset database
echo "📋 Resetting database..."
docker exec inventory-postgres psql -U postgres -d inventory_db -q <<EOF
UPDATE products SET reserved_quantity = 0 WHERE product_id = 'prd_laptop_001';
TRUNCATE processed_orders, inventory_logs;
EOF

# Check initial stock
INITIAL_STOCK=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT stock_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

echo "✓ Initial stock: ${INITIAL_STOCK} laptops"
echo ""

# Test parameters
PRODUCT_ID="prd_laptop_001"
QUANTITY=10
CONCURRENT_REQUESTS=20

echo "🚀 Test Configuration:"
echo "   Product:      ${PRODUCT_ID}"
echo "   Quantity:     ${QUANTITY} per order"
echo "   Concurrency:  ${CONCURRENT_REQUESTS} simultaneous requests"
echo "   Total demand: $((QUANTITY * CONCURRENT_REQUESTS)) units"
echo ""
echo "⏳ Sending ${CONCURRENT_REQUESTS} concurrent requests (same order details)..."

# Generate a fixed timestamp for "same order" simulation
FIXED_TIMESTAMP=$(date +%s)

# Launch concurrent requests
for i in $(seq 1 $CONCURRENT_REQUESTS); do
  {
    curl -s -X POST http://localhost:8081/orders \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"usr_race_test\",
        \"product_id\": \"${PRODUCT_ID}\",
        \"quantity\": ${QUANTITY}
      }" > /dev/null
  } &
done

# Wait for all requests to complete
wait

echo "✓ All requests sent"
echo ""
echo "⏳ Waiting for consumer processing (8 seconds)..."
sleep 8

# Collect results
echo ""
echo "═══════════════════════════════════════════════════════"
echo "                    RESULTS ANALYSIS                    "
echo "═══════════════════════════════════════════════════════"
echo ""

# Count processed orders
PROCESSED_COUNT=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

# Get reserved quantity
RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT reserved_quantity FROM products WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

# Calculate expected vs actual
EXPECTED_RESERVED=$((QUANTITY * CONCURRENT_REQUESTS))
OVER_RESERVED=$((RESERVED - EXPECTED_RESERVED))

echo "📊 Processing Summary:"
echo "   Orders processed:  ${PROCESSED_COUNT}"
echo "   Expected reserved: ${EXPECTED_RESERVED} units (${CONCURRENT_REQUESTS} × ${QUANTITY})"
echo "   Actual reserved:   ${RESERVED} units"
echo ""

# Determine if race condition occurred
if [ "$RESERVED" -gt "$EXPECTED_RESERVED" ]; then
    EXTRA_PERCENT=$(( (OVER_RESERVED * 100) / EXPECTED_RESERVED ))
    echo -e "${RED}❌ RACE CONDITION DETECTED!${NC}"
    echo -e "${RED}   Over-reserved: ${OVER_RESERVED} units (${EXTRA_PERCENT}% excess)${NC}"
    echo ""
    echo "🔍 Root Cause Analysis:"
    echo "   1. No partition key → Orders distributed randomly"
    echo "   2. Same order → Different partitions"
    echo "   3. Different consumers → Process simultaneously"
    echo "   4. No idempotency check → Multiple deductions"
    echo ""
elif [ "$PROCESSED_COUNT" -gt "$CONCURRENT_REQUESTS" ]; then
    echo -e "${RED}❌ DUPLICATE PROCESSING DETECTED!${NC}"
    DUPES=$((PROCESSED_COUNT - CONCURRENT_REQUESTS))
    echo -e "${RED}   Duplicates: ${DUPES} orders${NC}"
    echo ""
else
    echo -e "${YELLOW}⚠️  No obvious race condition detected${NC}"
    echo "   This can happen if:"
    echo "   - Consumers are too slow (queue backed up)"
    echo "   - Not enough concurrent load"
    echo "   - Lucky timing (rare)"
    echo ""
    echo "   Recommendation: Run test again or increase concurrency"
fi

# Show detailed processing breakdown
echo "📋 Detailed Order Breakdown:"
docker exec inventory-postgres psql -U postgres -d inventory_db <<EOF
SELECT 
    order_id,
    quantity,
    processed_at,
    EXTRACT(EPOCH FROM (processed_at - LAG(processed_at) OVER (ORDER BY processed_at))) * 1000 as gap_ms
FROM processed_orders 
WHERE product_id = '${PRODUCT_ID}'
ORDER BY processed_at
LIMIT 10;
EOF

# Check inventory consistency
echo ""
echo "📦 Inventory Status:"
docker exec inventory-postgres psql -U postgres -d inventory_db <<EOF
SELECT 
    product_id,
    stock_quantity,
    reserved_quantity,
    available_quantity,
    CASE 
        WHEN reserved_quantity > stock_quantity THEN 'OVERSOLD ❌'
        ELSE 'OK ✓'
    END as status
FROM products 
WHERE product_id = '${PRODUCT_ID}';
EOF

echo ""
echo "═══════════════════════════════════════════════════════"
echo "                    TEST COMPLETE                       "
echo "═══════════════════════════════════════════════════════"
