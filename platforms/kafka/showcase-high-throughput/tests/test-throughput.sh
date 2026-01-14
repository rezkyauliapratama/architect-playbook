#!/bin/bash

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  TEST 2: THROUGHPUT MEASUREMENT                       ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

# ============================================================
# TEST PARAMETERS
# ============================================================
TOTAL_MESSAGES=10000        # Increase to 10K for better measurement
PRODUCT_ID="prd_keyboard_003"
BATCH_SIZE=100              # Send in batches to prevent client overwhelming
PROCESSING_WAIT=30          # Wait for queue to drain

echo "📊 Test Configuration:"
echo "   Total messages:    ${TOTAL_MESSAGES}"
echo "   Product ID:        ${PRODUCT_ID}"
echo "   Batch size:        ${BATCH_SIZE}"
echo "   Processing wait:   ${PROCESSING_WAIT}s"
echo ""

# ============================================================
# PHASE 1: ENVIRONMENT RESET
# ============================================================
echo "📋 Resetting database..."
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    UPDATE products SET reserved_quantity = 0, stock_quantity = 999999 WHERE product_id = '${PRODUCT_ID}';
    TRUNCATE processed_orders CASCADE;
    TRUNCATE inventory_logs CASCADE;
"
echo -e "${GREEN}✓ Database reset complete${NC}"
echo ""

# ============================================================
# PHASE 2: THROUGHPUT TEST EXECUTION
# ============================================================
echo "🚀 Starting throughput test..."
echo "   Sending ${TOTAL_MESSAGES} orders as fast as possible..."
echo ""

# Record start time with millisecond precision
START_TIME=$(date +%s%N)
START_TIME_SEC=$(date +%s)

# Counter for progress display
SENT_COUNT=0

# Send messages in batches
for i in $(seq 1 $TOTAL_MESSAGES); do
  USER_ID="usr_throughput_${i}"
  
  # Send HTTP request in background (non-blocking)
  curl -s -X POST http://localhost:8081/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"${USER_ID}\",
      \"product_id\": \"${PRODUCT_ID}\",
      \"quantity\": 1
    }" > /dev/null 2>&1 &
  
  ((SENT_COUNT++))
  
  # Wait every BATCH_SIZE messages to:
  # 1. Prevent overwhelming the system with too many concurrent connections
  # 2. Allow background jobs to complete
  # 3. Provide progress feedback
  if [ $((i % BATCH_SIZE)) -eq 0 ]; then
    wait  # Wait for all background curl jobs in this batch
    
    # Calculate and display progress
    PERCENT=$((i * 100 / TOTAL_MESSAGES))
    echo -e "  ${BLUE}Progress: ${i}/${TOTAL_MESSAGES} (${PERCENT}%)${NC}"
  fi
done

# Wait for all remaining background jobs
wait

# Record end time
END_TIME=$(date +%s%N)
END_TIME_SEC=$(date +%s)

# Calculate send duration
SEND_DURATION_NS=$((END_TIME - START_TIME))
SEND_DURATION_MS=$((SEND_DURATION_NS / 1000000))
SEND_DURATION_SEC=$((END_TIME_SEC - START_TIME_SEC))

echo ""
echo -e "${GREEN}✓ All ${TOTAL_MESSAGES} requests sent${NC}"
echo "   Send duration: ${SEND_DURATION_SEC} seconds (${SEND_DURATION_MS}ms)"
echo ""

# ============================================================
# PHASE 3: WAIT FOR PROCESSING
# ============================================================
echo "⏳ Waiting for consumer processing (${PROCESSING_WAIT}s)..."

# Poll processing status every 5 seconds
for ((i=0; i<PROCESSING_WAIT; i+=5)); do
  sleep 5
  
  # Query current processed count
  CURRENT_PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
  " | tr -d ' ')
  
  PERCENT=$((CURRENT_PROCESSED * 100 / TOTAL_MESSAGES))
  echo "  Processed: ${CURRENT_PROCESSED}/${TOTAL_MESSAGES} (${PERCENT}%)"
done

echo ""

# ============================================================
# PHASE 4: COLLECT RESULTS
# ============================================================
echo "═══════════════════════════════════════════════════════"
echo "                   RESULTS ANALYSIS                     "
echo "═══════════════════════════════════════════════════════"
echo ""

# Query final processed count
PROCESSED=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(*) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

# Query unique users
UNIQUE_USERS=$(docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
    SELECT COUNT(DISTINCT user_id) FROM processed_orders WHERE product_id = '${PRODUCT_ID}';
" | tr -d ' ')

# Check for duplicates
DUPLICATES=$((PROCESSED - UNIQUE_USERS))

# Calculate total end-to-end time
TOTAL_TIME=$((END_TIME_SEC - START_TIME_SEC + PROCESSING_WAIT))

# ============================================================
# PHASE 5: METRICS CALCULATION
# ============================================================

# Throughput calculations
SEND_THROUGHPUT=$((TOTAL_MESSAGES / SEND_DURATION_SEC))
END_TO_END_THROUGHPUT=$((PROCESSED / TOTAL_TIME))
PROCESSING_THROUGHPUT=$((PROCESSED / PROCESSING_WAIT))

# Latency calculations
AVG_SEND_LATENCY_MS=$((SEND_DURATION_MS / TOTAL_MESSAGES))

# Success rate
SUCCESS_RATE=$((PROCESSED * 100 / TOTAL_MESSAGES))

# Message loss
if [ "$PROCESSED" -lt "$TOTAL_MESSAGES" ]; then
  LOST=$((TOTAL_MESSAGES - PROCESSED))
  LOSS_RATE=$((LOST * 100 / TOTAL_MESSAGES))
else
  LOST=0
  LOSS_RATE=0
fi

# ============================================================
# PHASE 6: DISPLAY RESULTS
# ============================================================

echo "📊 Summary Metrics:"
echo "   Messages Sent:        ${TOTAL_MESSAGES}"
echo "   Messages Processed:   ${PROCESSED}"
echo "   Unique Users:         ${UNIQUE_USERS}"
echo "   Duplicates:           ${DUPLICATES}"
echo "   Success Rate:         ${SUCCESS_RATE}%"
echo ""

echo "⏱️  Timing:"
echo "   Send Duration:        ${SEND_DURATION_SEC}s"
echo "   Processing Duration:  ${PROCESSING_WAIT}s"
echo "   Total End-to-End:     ${TOTAL_TIME}s"
echo ""

echo "🚀 Throughput:"
echo "   Send Throughput:      ${SEND_THROUGHPUT} msg/s"
echo "   Processing Throughput: ${PROCESSING_THROUGHPUT} msg/s"
echo "   End-to-End Throughput: ${END_TO_END_THROUGHPUT} msg/s"
echo ""

echo "📈 Latency:"
echo "   Avg Send Latency:     ${AVG_SEND_LATENCY_MS}ms per message"
echo ""

# ============================================================
# PHASE 7: FAILURE DETECTION
# ============================================================

if [ "$LOST" -gt 0 ]; then
  echo -e "${RED}❌ MESSAGE LOSS DETECTED!${NC}"
  echo -e "${RED}   Lost: ${LOST} messages (${LOSS_RATE}%)${NC}"
  echo ""
fi

if [ "$DUPLICATES" -gt 0 ]; then
  echo -e "${RED}❌ DUPLICATE PROCESSING DETECTED!${NC}"
  echo -e "${RED}   Duplicates: ${DUPLICATES} messages${NC}"
  echo ""
fi

# ============================================================
# PHASE 8: PERFORMANCE ANALYSIS
# ============================================================

echo "📉 Performance Analysis:"
echo ""

# Calculate theoretical capacity (rough estimate)
THEORETICAL_CAPACITY=10000  # messages per second (conservative estimate)
CAPACITY_UTILIZATION=$((END_TO_END_THROUGHPUT * 100 / THEORETICAL_CAPACITY))

echo "   Theoretical Capacity:  ${THEORETICAL_CAPACITY} msg/s"
echo "   Actual Throughput:     ${END_TO_END_THROUGHPUT} msg/s"
echo "   Capacity Utilization:  ${CAPACITY_UTILIZATION}%"
echo ""

if [ "$CAPACITY_UTILIZATION" -lt 10 ]; then
  echo -e "${RED}⚠️  SEVERE UNDERUTILIZATION (<10%)${NC}"
  echo "   Indicates major configuration bottlenecks:"
  echo "   - No message batching (linger.ms=0)"
  echo "   - No compression enabled"
  echo "   - Sequential consumer processing"
  echo "   - Individual network calls per message"
elif [ "$CAPACITY_UTILIZATION" -lt 30 ]; then
  echo -e "${YELLOW}⚠️  LOW UTILIZATION (<30%)${NC}"
  echo "   System not efficiently using available capacity"
else
  echo -e "${GREEN}✓ Acceptable utilization${NC}"
fi

echo ""

# ============================================================
# PHASE 9: BOTTLENECK IDENTIFICATION
# ============================================================

echo "🔍 Bottleneck Identification:"
echo ""

# Estimate network overhead
EXPECTED_BATCH_COUNT=$((TOTAL_MESSAGES / 100))  # If batching was optimal
ACTUAL_NETWORK_CALLS=$((TOTAL_MESSAGES))  # Current: 1 call per message
NETWORK_OVERHEAD=$((ACTUAL_NETWORK_CALLS - EXPECTED_BATCH_COUNT))

echo "   Expected Network Calls (with batching): ${EXPECTED_BATCH_COUNT}"
echo "   Actual Network Calls (no batching):     ${ACTUAL_NETWORK_CALLS}"
echo "   Network Overhead:                       ${NETWORK_OVERHEAD} extra calls"
echo ""

if [ "$END_TO_END_THROUGHPUT" -lt 200 ]; then
  echo -e "${RED}❌ CRITICAL THROUGHPUT ISSUE${NC}"
  echo "   Root causes:"
  echo "   1. Producer: No batching (batch.size=16KB, linger.ms=0)"
  echo "   2. Producer: No compression (compression.type=none)"
  echo "   3. Consumer: Sequential processing (1 message at a time)"
  echo "   4. Consumer: Individual database writes (no bulk insert)"
  echo ""
  echo "   Impact:"
  echo "   - 1 network round-trip per message"
  echo "   - Full message payload transmitted"
  echo "   - Consumer blocked on each message"
  echo "   - Database connection overhead per write"
fi

echo ""
echo "═══════════════════════════════════════════════════════"
echo ""

# ============================================================
# PHASE 10: DETAILED STATISTICS (OPTIONAL)
# ============================================================

echo "📋 Detailed Statistics:"
echo ""

# Query processing time distribution (if timestamp available)
docker exec inventory-postgres psql -U postgres -d inventory_db -t -c "
SELECT 
  COUNT(*) as total_orders,
  MIN(quantity) as min_quantity,
  MAX(quantity) as max_quantity,
  AVG(quantity)::numeric(10,2) as avg_quantity
FROM processed_orders
WHERE product_id = '${PRODUCT_ID}';
"

echo ""
echo "═══════════════════════════════════════════════════════"
