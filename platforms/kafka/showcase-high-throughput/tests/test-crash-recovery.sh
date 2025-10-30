#!/bin/bash

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  TEST 5: CONSUMER CRASH RECOVERY (FIXED)             ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

VERSION=${1:-baseline}

if [ "$VERSION" == "baseline" ]; then
    echo "🔴 Testing BASELINE (Auto-commit)"
    SERVICE_NAME="showcase-high-throughput-inventory-service-1 showcase-high-throughput-inventory-service-2 showcase-high-throughput-inventory-service-3"
    GROUP_ID="inventory-group-baseline"
else
    echo "🟢 Testing IMPROVED (Manual commit)"
    SERVICE_NAME="inventory-service-improved"
    GROUP_ID="inventory-group-improved"
fi
````
echo "Expected behavior:"
if [ "$VERSION" == "baseline" ]; then
    echo "   - Auto-commit every 5 seconds"
    echo "   - Crash will cause data loss or duplicate processing"
else
    echo "   - Manual commit after processing"
    echo "   - No data loss, idempotency handles duplicates"
fi
echo ""

echo "🔧 Resetting environment..."

# 1. Reset database
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
TRUNCATE TABLE processed_orders, inventory_logs CASCADE;
UPDATE products SET reserved_quantity = 0, stock_quantity = 20000;
"
echo "   ✓ Database reset"

# 2. Stop consumer to empty the group
echo "   Stopping consumer..."
docker stop $SERVICE_NAME 2>/dev/null || true
sleep 5  # Wait for consumer to fully disconnect

# 3. Delete consumer group (now it's empty)
echo "   Deleting consumer group..."
docker exec redpanda-0 rpk group delete $GROUP_ID \
  --brokers redpanda-0:9092,redpanda-1:9092,redpanda-2:9092 \
  2>&1 | grep -q "COORDINATOR_NOT_AVAILABLE\|NOT_FOUND" || echo "   ✓ Group deleted"

# Alternative: If delete still fails, just reset offsets
if docker exec redpanda-0 rpk group describe $GROUP_ID \
   --brokers redpanda-0:9092 2>&1 | grep -q "State.*Empty"; then
    echo "   ✓ Group is empty"
fi

# 4. Start consumer (creates new group)
echo "   Starting consumer..."
docker start $SERVICE_NAME 2>/dev/null || \
docker-compose up -d $SERVICE_NAME


sleep 5``

echo "✓ Environment reset"
echo ""

TOTAL_ORDERS=3000


# Check initial state
echo "📊 Initial state:"
docker exec redpanda-0 rpk group describe $GROUP_ID 2>&1 | grep -E "GROUP|STATE|TOTAL-LAG" || echo "   Consumer group not active"
echo ""

echo "🚀 Test Configuration:"
echo "   Orders to send:    $TOTAL_ORDERS"
echo "   Quantity per order: 5 units"
echo "   Expected total:    3000 units reserved"
echo "   Crash timing:      After 50-100 orders processed"
echo ""

# Send messages in background (slow pace to allow crash)
echo "⏳ Sending 3000 $TOTAL_ORDERS (in background)..."
{
    for i in $(seq 1 $TOTAL_ORDERS); do
        curl -s -X POST http://localhost:8081/orders \
          -H "Content-Type: application/json" \
          -d "{
            \"user_id\": \"usr_crash_test_${i}\",
            \"product_id\": \"prd_laptop_001\",
            \"quantity\": 5
          }" > /dev/null 2>&1
        
        # Small delay to prevent overwhelming
        [ $((i % 50)) -eq 0 ] && echo "   Sent $i..." && sleep 0.5
    done
    echo "✓ All $TOTAL_ORDERS orders sent" > /tmp/orders_sent.flag
} &

SEND_PID=$!

echo "   Orders being sent in background..."
echo ""
PROCESSED_BEFORE=0
# Monitor processing
echo "📊 Monitoring processing..."
for i in {1..15}; do
    PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -A -c "
        SELECT COUNT(*)::int FROM processed_orders;
    " 2>&1 | grep -E '^[0-9]+$' || echo "0")
    
    echo "   T+${i}s: $PROCESSED orders processed"
    
    # Crash when we have 50-150 processed
    if [ "$PROCESSED" -ge 150 ] && [ "$PROCESSED" -le 1000 ]; then
        echo ""
        echo "💥 CRASHING CONSUMER NOW (processed: $PROCESSED)..."
        
        PROCESSED_BEFORE=$PROCESSED
        
        # DEBUG: Verify variable is set
        echo "   [DEBUG] PROCESSED_BEFORE set to: $PROCESSED_BEFORE"
        
        # Get current offsets BEFORE crash
        echo "   Checking offsets before crash..."
        docker exec redpanda-0 rpk group describe $GROUP_ID 2>/dev/null | grep -E "CURRENT-OFFSET|LAG" || true
        
        # KILL -9 (hard kill, no cleanup)
        docker kill -s KILL $SERVICE_NAME 2>/dev/null || docker kill $SERVICE_NAME
        sleep 2
        echo "   ✓ Consumer killed"
        break
    fi
    
    sleep 1
done

# Wait for all orders to be sent
echo ""
echo "⏳ Waiting for all orders to be sent..."
wait $SEND_PID 2>/dev/null || true
sleep 2

# Check sent count
SENT_COUNT=$(docker exec redpanda-0 rpk topic describe orders -p 2>/dev/null | grep "high watermark" | head -1 | awk '{print $NF}' || echo "3000")
echo "   ✓ Orders in Kafka: ~$SENT_COUNT"
echo ""

# Show state BEFORE restart
echo "📊 State BEFORE restart:"
echo "   Processed before crash: $PROCESSED_BEFORE"
docker exec redpanda-0 rpk group describe $GROUP_ID 2>&1 | grep -A 20 "PARTITION" || echo "   Group state unknown"
echo ""

if [ "$VERSION" == "baseline" ]; then
    echo "⚠️  BASELINE: Auto-commit may have committed offsets for unprocessed messages"
    echo "   Risk: Messages between last commit and crash will be LOST"
else
    echo "✅ IMPROVED: Manual commit ensures offsets match processing"
    echo "   Expectation: All messages will be reprocessed from last commit"
fi
echo ""

# Restart consumer
echo "🔄 Restarting consumer..."
docker restart $SERVICE_NAME > /dev/null 2>&1
sleep 8
echo "   ✓ Consumer restarted"
echo ""

# Wait for recovery
echo "⏳ Waiting for recovery processing (20 seconds)..."
for i in {1..20}; do
    sleep 1
    CURRENT=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
        SELECT COUNT(*) FROM processed_orders;
    " 2>/dev/null | tr -d ' ' || echo "0")
    echo "   T+${i}s: $CURRENT orders processed"
done
echo ""


# Analysis
echo "═══════════════════════════════════════════════════════"
echo "                   RESULTS ANALYSIS                     "
echo "═══════════════════════════════════════════════════════"
echo ""

TOTAL_PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -A -c "
    SELECT COUNT(*) FROM processed_orders;
" | grep -E '^[0-9]+$')

UNIQUE_USERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -A -c "
    SELECT COUNT(DISTINCT user_id) FROM processed_orders;
" | grep -E '^[0-9]+$')

RESERVED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -A -c "
    SELECT reserved_quantity FROM products WHERE product_id = 'prd_laptop_001';
" | grep -E '^[0-9]+$')

EXPECTED_RESERVED=$((TOTAL_ORDERS * 5))
PROCESSED_AFTER=$((TOTAL_PROCESSED - PROCESSED_BEFORE))
DATA_LOSS=$((TOTAL_ORDERS - UNIQUE_USERS))
DUPLICATES=$((TOTAL_PROCESSED - UNIQUE_USERS))
OVER_RESERVED=$((RESERVED - EXPECTED_RESERVED))

echo "📊 Processing Summary:"
echo "   Orders sent:              $TOTAL_ORDERS"
echo "   Processed before crash:   $PROCESSED_BEFORE"
echo "   Processed after restart:  $PROCESSED_AFTER"
echo "   Total processed:          $TOTAL_PROCESSED"
echo "   Unique users:             $UNIQUE_USERS"
echo ""

echo "📦 Inventory:"
echo "   Expected reservation:     $EXPECTED_RESERVED units ($TOTAL_ORDERS * 5)"
echo "   Actual reservation:       $RESERVED units"
if [ "$OVER_RESERVED" -ne 0 ]; then
    echo "   Over-reservation:         $OVER_RESERVED units"
fi
echo ""

# Critical analysis
HAS_ISSUES=false

if [ "$DATA_LOSS" -gt 0 ]; then
    HAS_ISSUES=true
    LOSS_PCT=$(awk "BEGIN {printf \"%.2f\", ($DATA_LOSS * 100.0 / $TOTAL_ORDERS)}")
    echo -e "${RED}❌ DATA LOSS DETECTED!${NC}"
    echo -e "${RED}   Missing orders: $DATA_LOSS (${LOSS_PCT}%)${NC}"
    echo ""
    echo "🔍 Root Cause:"
    echo "   - Auto-commit committed offsets ahead of processing"
    echo "   - Crash occurred after commit"
    echo "   - On restart, consumer skipped uncommitted messages"
    echo "   - $DATA_LOSS orders permanently lost"
    echo ""
fi

if [ "$DUPLICATES" -gt 0 ]; then
    HAS_ISSUES=true
    DUP_PCT=$(awk "BEGIN {printf \"%.2f\", ($DUPLICATES * 100.0 / $UNIQUE_USERS)}")
    echo -e "${YELLOW}⚠️  DUPLICATE PROCESSING: $DUPLICATES (${DUP_PCT}%)${NC}"
    echo ""
fi

# ⚠️ CRITICAL FIX: Check over-reservation regardless of duplicates
if [ "$OVER_RESERVED" -gt 0 ]; then
    HAS_ISSUES=true
    OVER_PCT=$(awk "BEGIN {printf \"%.2f\", ($OVER_RESERVED * 100.0 / $EXPECTED_RESERVED)}")
    echo -e "${RED}❌ INVENTORY INCONSISTENCY!${NC}"
    echo -e "${RED}   Over-reserved: $OVER_RESERVED units (${OVER_PCT}%)${NC}"
    echo ""
    echo "🔍 Root Cause:"
    echo "   Timeline reconstruction:"
    echo "   1. Processed $PROCESSED_BEFORE orders before crash"
    echo "   2. Auto-commit ran during processing (commits ahead)"
    echo "   3. Consumer crashed"
    echo "   4. On restart, replayed from LAST PROCESSED offset"
    echo "   5. Some messages processed TWICE"
    echo "   6. No idempotency check → Double inventory reservation"
    echo ""
    echo "   Approximate duplicate orders: ~$((OVER_RESERVED / 5))"
    echo "   (${OVER_RESERVED} units ÷ 5 units/order)"
    echo ""
elif [ "$OVER_RESERVED" -lt 0 ]; then
    HAS_ISSUES=true
    UNDER_RESERVED=$((0 - OVER_RESERVED))
    UNDER_PCT=$(awk "BEGIN {printf \"%.2f\", ($UNDER_RESERVED * 100.0 / $EXPECTED_RESERVED)}")
    echo -e "${RED}❌ UNDER-RESERVATION!${NC}"
    echo -e "${RED}   Under-reserved: $UNDER_RESERVED units (${UNDER_PCT}%)${NC}"
    echo ""
    echo "🔍 This indicates data loss (orders not processed)"
    echo ""
fi

# Final verdict
echo "═══════════════════════════════════════════════════════"
echo "                    TEST COMPLETE                       "
echo "═══════════════════════════════════════════════════════"
echo ""

if [ "$VERSION" == "baseline" ]; then
    if [ "$HAS_ISSUES" = true ]; then
        echo -e "${RED}Result: BASELINE FAILED ✗${NC}"
        echo ""
        echo "Problems detected:"
        [ "$DATA_LOSS" -gt 0 ] && echo "   ❌ Data loss: $DATA_LOSS orders"
        [ "$OVER_RESERVED" -gt 0 ] && echo "   ❌ Over-reservation: $OVER_RESERVED units"
        [ "$DUPLICATES" -gt 0 ] && echo "   ⚠️  Duplicate records: $DUPLICATES"
        echo ""
        echo "Root Cause: Auto-commit"
        echo "   - Commits offsets before processing completes"
        echo "   - No idempotency check"
        echo "   - Crash creates inconsistency window"
        echo ""
        echo "Business Impact:"
        [ "$DATA_LOSS" -gt 0 ] && echo "   - Lost revenue from $DATA_LOSS unprocessed orders"
        [ "$OVER_RESERVED" -gt 0 ] && echo "   - Inventory mismatch requires manual reconciliation"
        echo "   - Customer service tickets for missing/duplicate orders"
    else
        echo -e "${YELLOW}Result: No issues detected${NC}"
        echo "   (Crash timing was lucky)"
    fi
else
    if [ "$DATA_LOSS" -eq 0 ] && [ "$OVER_RESERVED" -eq 0 ]; then
        echo -e "${GREEN}Result: IMPROVED VERSION PASSED ✓${NC}"
        echo ""
        echo "Guarantees delivered:"
        echo "   ✅ Zero data loss"
        echo "   ✅ Inventory consistent"
        echo "   ✅ Manual commit working"
        [ "$DUPLICATES" -gt 0 ] && echo "   ✅ Idempotency handled $DUPLICATES replays"
    else
        echo -e "${RED}Result: IMPROVED VERSION FAILED ✗${NC}"
        [ "$DATA_LOSS" -gt 0 ] && echo "   ❌ Data loss: $DATA_LOSS"
        [ "$OVER_RESERVED" -ne 0 ] && echo "   ❌ Inventory inconsistent: $OVER_RESERVED"
    fi
fi

echo ""