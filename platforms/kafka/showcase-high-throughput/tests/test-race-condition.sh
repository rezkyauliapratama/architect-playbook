#!/bin/bash

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  TEST 1: RACE CONDITION (ULTRA-AGGRESSIVE)            ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test parameters - AGGRESSIVE
INITIAL_STOCK=300
PRODUCT_ID="prd_laptop_001"
CONCURRENT_ORDERS=100
QUANTITY=20
TOTAL_DEMAND=$((QUANTITY * CONCURRENT_ORDERS))
RETRIES=3  # Send each order 3 times

# Reset
echo "📋 Resetting database..."
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
UPDATE products SET reserved_quantity = 0, stock_quantity = ${INITIAL_STOCK} WHERE product_id = 'prd_laptop_001';
DELETE FROM processed_orders;
"

# CRITICAL: Lower stock to GUARANTEE overselling
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
UPDATE products SET stock_quantity = ${INITIAL_STOCK}  WHERE product_id = 'prd_laptop_001';
"

STOCK=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT stock_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

echo -e "${GREEN}✓ Stock set to: ${STOCK} units${NC}"
echo ""


echo "🚀 Test Configuration (AGGRESSIVE):"
echo "   Product:          ${PRODUCT_ID}"
echo "   Stock available:  ${STOCK} units"
echo "   Quantity/order:   ${QUANTITY} units"
echo "   Concurrent orders: ${CONCURRENT_ORDERS}"
echo "   Total demand:     ${TOTAL_DEMAND} units"
echo "   Expected result:  OVERSELLING (demand > stock)"
echo ""


echo "⏳ Sending ${CONCURRENT_ORDERS} orders as fast as possible..."

# Send ALL at once (no batching)
for i in $(seq 1 $CONCURRENT_ORDERS); do
  USER_ID="usr_duplicate_${i}"

#  echo "  User ${USER_ID}: Sending order ${RETRIES}x (retry simulation)..."
  
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
echo -e "${GREEN}✓ All ${CONCURRENT_ORDERS} orders sent${NC}"
echo ""

echo "⏳ Waiting for processing (20 seconds)..."
sleep 20

# Results
echo ""
echo "═══════════════════════════════════════════════════════"
echo "                   RESULTS ANALYSIS                     "
echo "═══════════════════════════════════════════════════════"
echo ""

PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT reserved_quantity FROM products WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

UNIQUE_USERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(DISTINCT user_id) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

echo "📊 Results:"
echo "   Orders sent:       ${CONCURRENT_ORDERS}"
echo "   Orders processed:  ${PROCESSED}"
echo "   Unique users:      ${UNIQUE_USERS}"
echo ""
echo "📦 Inventory:"
echo "   Stock available:   ${STOCK} units"
echo "   Total demand:      ${TOTAL_DEMAND} units"
echo "   Actual reserved:   ${RESERVED} units"
echo ""

# Analysis
EXPECTED_RESERVED=$((PROCESSED * QUANTITY))
OVER_RESERVED=$((RESERVED - STOCK))

if [ "$RESERVED" -gt "$STOCK" ]; then
    OVERSELL_PERCENT=$(( (OVER_RESERVED * INITIAL_STOCK) / STOCK ))
    echo -e "${RED}❌ OVERSELLING DETECTED!${NC}"
    echo -e "${RED}   Oversold: ${OVER_RESERVED} units (${OVERSELL_PERCENT}% over stock)${NC}"
    echo ""
    echo "🔍 Root Cause:"
    echo "   Multiple consumers read stock=${STOCK} simultaneously"
    echo "   All think enough stock available"
    echo "   All reserve → Total reserved (${RESERVED}) > Available (${STOCK})"
    echo ""
    echo -e "${RED}Result: BASELINE SYSTEM FAILED - CRITICAL BUG${NC}"
elif [ "$PROCESSED" -ne "$UNIQUE_USERS" ]; then
    DUPES=$((PROCESSED - UNIQUE_USERS))
    echo -e "${RED}❌ DUPLICATE PROCESSING!${NC}"
    echo -e "${RED}   Duplicates: ${DUPES} orders${NC}"
else
    echo -e "${YELLOW}⚠️  No race condition reproduced${NC}"
    echo "   Try: Rebuild baseline service without idempotency check"
fi

# Show overselling in database
echo ""
echo "📦 Final State (OVERSOLD):"
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
SELECT 
    product_id,
    stock_quantity as stock,
    reserved_quantity as reserved,
    (reserved_quantity - stock_quantity) as oversold,
    CASE 
        WHEN reserved_quantity > stock_quantity THEN '❌ OVERSOLD'
        ELSE '✓ OK'
    END as status
FROM products 
WHERE product_id = '${PRODUCT_ID}';
"

echo ""
echo "═══════════════════════════════════════════════════════"
