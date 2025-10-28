#!/bin/bash

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  TEST 5: CONSUMER CRASH RECOVERY                      ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

VERSION=${1:-baseline}  # baseline or improved

if [ "$VERSION" == "baseline" ]; then
    echo "🔴 Testing BASELINE (Auto-commit)"
    echo "Expected: Data loss or duplicate processing on crash"
    SERVICE_NAME="showcase-high-throughput-inventory-service-1 showcase-high-throughput-inventory-service-2 showcase-high-throughput-inventory-service-3"
else
    echo "🟢 Testing IMPROVED (Manual commit)"
    echo "Expected: No data loss, no duplicates"
    SERVICE_NAME="inventory-service-improved"
fi

echo ""

# Reset
echo "📋 Resetting database..."
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
UPDATE products SET reserved_quantity = 0, stock_quantity = 10000;
DELETE FROM processed_orders;
DELETE FROM inventory_logs;
"

echo "✓ Database reset"
echo ""

# Check consumer lag before test
echo "📊 Initial consumer state:"
docker exec redpanda-0 rpk group describe inventory-group-${VERSION} 2>/dev/null || echo "   Consumer group not yet active"
echo ""

echo "🚀 Test Scenario:"
echo "   1. Send 1000 orders"
echo "   2. Wait for partial processing (5 seconds)"
echo "   3. CRASH consumer (kill -9)"
echo "   4. Restart consumer"
echo "   5. Check for data loss or duplicates"
echo ""

# Send orders
echo "⏳ Sending 1000 orders..."
for i in {1..1000}; do
  {
    curl -s -X POST http://localhost:8081/orders \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"usr_crash_${i}\",
        \"product_id\": \"prd_laptop_001\",
        \"quantity\": 5
      }" > /dev/null
  } &
  
  if [ $((i % 20)) -eq 0 ]; then
    wait
  fi
done

wait
echo "✓ 1000 orders sent"
echo ""

# Wait for partial processing
echo "⏳ Waiting 5 seconds for partial processing..."
sleep 5

# Check how many processed before crash
PROCESSED_BEFORE=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

echo "   Processed before crash: ${PROCESSED_BEFORE} orders"
echo ""

# Get consumer lag before crash
echo "📊 Consumer state before crash:"
docker exec redpanda-0 rpk group describe inventory-group-${VERSION} 2>/dev/null || true
echo ""

# CRASH THE CONSUMER
echo "💥 CRASHING consumer (kill -9)..."
docker kill -s KILL ${SERVICE_NAME} 2>/dev/null || true
sleep 2
echo "   Consumer killed"
echo ""

# Check what was committed
if [ "$VERSION" == "baseline" ]; then
    echo "⚠️  BASELINE: Auto-commit may have committed offsets for unprocessed messages"
    echo "   → Data loss expected"
else
    echo "✅ IMPROVED: Manual commit only commits after processing"
    echo "   → No data loss expected"
fi
echo ""

# Restart consumer
echo "🔄 Restarting consumer..."
docker restart ${SERVICE_NAME} > /dev/null 2>&1
sleep 8
echo "   Consumer restarted"
echo ""

# Wait for recovery processing
echo "⏳ Waiting 10 seconds for recovery processing..."
sleep 10

# Collect results
echo ""
echo "═══════════════════════════════════════════════════════"
echo "                   RESULTS ANALYSIS                     "
echo "═══════════════════════════════════════════════════════"
echo ""

TOTAL_PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders;
" | tr -d ' ')

UNIQUE_USERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(DISTINCT user_id) FROM processed_orders;
" | tr -d ' ')

RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT reserved_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | tr -d ' ')

PROCESSED_AFTER_CRASH=$((TOTAL_PROCESSED - PROCESSED_BEFORE))

# Calculate metrics
EXPECTED_RESERVED=$((UNIQUE_USERS * 5))
DATA_LOSS=$((100 - UNIQUE_USERS))
DUPLICATES=$((TOTAL_PROCESSED - UNIQUE_USERS))

echo "📊 Processing Summary:"
echo "   Orders sent:             100"
echo "   Processed before crash:  ${PROCESSED_BEFORE}"
echo "   Processed after restart: ${PROCESSED_AFTER_CRASH}"
echo "   Total processed:         ${TOTAL_PROCESSED}"
echo "   Unique users:            ${UNIQUE_USERS}"
echo ""

echo "📦 Inventory:"
echo "   Expected reservation:    500 units (100 × 5)"
echo "   Actual reservation:      ${RESERVED} units"
echo ""

# Analysis
if [ "$DATA_LOSS" -gt 0 ]; then
    DATA_LOSS_PERCENT=$(( (DATA_LOSS * 100) / 100 ))
    echo -e "${RED}❌ DATA LOSS DETECTED!${NC}"
    echo -e "${RED}   Lost orders: ${DATA_LOSS} (${DATA_LOSS_PERCENT}%)${NC}"
    echo ""
    echo "🔍 Root Cause:"
    if [ "$VERSION" == "baseline" ]; then
        echo "   1. Consumer auto-commits offsets every 5 seconds"
        echo "   2. Crash occurred between auto-commit and processing"
        echo "   3. On restart, consumer resumed from committed offset"
        echo "   4. Messages in the gap were never processed"
        echo "   5. Result: ${DATA_LOSS} orders lost"
    else
        echo "   ⚠️  Unexpected: Improved version should not lose data"
        echo "   Check manual commit implementation"
    fi
    echo ""
fi

if [ "$DUPLICATES" -gt 0 ]; then
    DUPLICATE_PERCENT=$(( (DUPLICATES * 100) / UNIQUE_USERS ))
    echo -e "${RED}❌ DUPLICATE PROCESSING DETECTED!${NC}"
    echo -e "${RED}   Duplicate records: ${DUPLICATES} (${DUPLICATE_PERCENT}%)${NC}"
    echo ""
    echo "🔍 Root Cause:"
    if [ "$VERSION" == "baseline" ]; then
        echo "   1. Consumer processed messages"
        echo "   2. Crash occurred before auto-commit"
        echo "   3. On restart, consumer replayed from last commit"
        echo "   4. Messages processed again"
        echo "   5. Result: ${DUPLICATES} duplicate records"
    else
        echo "   Note: Duplicates in improved version are expected if:"
        echo "   - Messages processed but not yet committed when crashed"
        echo "   - On restart, messages replayed (at-least-once)"
        echo "   - Idempotency check prevents duplicate effects"
        echo ""
        echo "   Checking if duplicates had actual impact..."
        
        # Check if duplicates caused over-reservation
        OVER_RESERVED=$((RESERVED - EXPECTED_RESERVED))
        if [ "$OVER_RESERVED" -gt 0 ]; then
            echo -e "${RED}   ❌ Duplicates caused over-reservation: ${OVER_RESERVED} units${NC}"
        else
            echo -e "${GREEN}   ✅ Duplicates caught by idempotency check${NC}"
            echo "      (No inventory impact)"
        fi
    fi
    echo ""
fi

if [ "$DATA_LOSS" -eq 0 ] && [ "$DUPLICATES" -eq 0 ]; then
    echo -e "${GREEN}✅ PERFECT RECOVERY!${NC}"
    echo "   - No data loss"
    echo "   - No duplicate processing"
    echo "   - All 100 orders processed exactly once"
    echo ""
    if [ "$VERSION" == "improved" ]; then
        echo "   Pattern 3 working correctly:"
        echo "   ✓ Manual commit"
        echo "   ✓ Idempotency check"
        echo "   ✓ Graceful recovery"
    fi
fi

# Show consumer lag after recovery
echo "📊 Consumer state after recovery:"
docker exec redpanda-0 rpk group describe inventory-group-${VERSION} 2>/dev/null || true
echo ""

# Show timing analysis
echo "⏱️  Processing Timeline:"
docker exec inventory-postgres psql -U postgres -d inventory_db <<EOF
SELECT 
    'Before crash' as phase,
    COUNT(*) as orders,
    MIN(processed_at) as first,
    MAX(processed_at) as last,
    EXTRACT(EPOCH FROM (MAX(processed_at) - MIN(processed_at))) as duration_sec
FROM processed_orders
WHERE processed_at <= (SELECT MIN(processed_at) + INTERVAL '5 seconds' FROM processed_orders)
UNION ALL
SELECT 
    'After restart' as phase,
    COUNT(*) as orders,
    MIN(processed_at) as first,
    MAX(processed_at) as last,
    EXTRACT(EPOCH FROM (MAX(processed_at) - MIN(processed_at))) as duration_sec
FROM processed_orders
WHERE processed_at > (SELECT MIN(processed_at) + INTERVAL '5 seconds' FROM processed_orders);
EOF

echo ""
echo "═══════════════════════════════════════════════════════"
echo "                    TEST COMPLETE                       "
echo "═══════════════════════════════════════════════════════"
echo ""

# Summary
if [ "$VERSION" == "baseline" ]; then
    if [ "$DATA_LOSS" -gt 0 ] || [ "$DUPLICATES" -gt 0 ]; then
        echo -e "${RED}Result: BASELINE FAILED${NC}"
        [ "$DATA_LOSS" -gt 0 ] && echo "   - Data loss: ${DATA_LOSS} orders"
        [ "$DUPLICATES" -gt 0 ] && echo "   - Duplicates: ${DUPLICATES} records"
    else
        echo -e "${YELLOW}Result: No issues detected (lucky timing)${NC}"
    fi
else
    if [ "$DATA_LOSS" -eq 0 ]; then
        echo -e "${GREEN}Result: IMPROVED VERSION PASSED${NC}"
        echo "   ✓ No data loss"
        echo "   ✓ Manual commit working"
        if [ "$DUPLICATES" -gt 0 ]; then
            OVER_RESERVED=$((RESERVED - EXPECTED_RESERVED))
            if [ "$OVER_RESERVED" -eq 0 ]; then
                echo "   ✓ Idempotency prevented duplicate effects"
            fi
        fi
    else
        echo -e "${RED}Result: IMPROVED VERSION FAILED${NC}"
        echo "   Check implementation"
    fi
fi

echo ""
