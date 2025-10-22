#!/bin/bash

echo "╔════════════════════════════════════════════╗"
echo "║  BASELINE (BEFORE) - COMPLETE TEST SUITE  ║"
echo "╚════════════════════════════════════════════╝"
echo ""

# Make all test scripts executable
chmod +x test-*.sh

# Run all tests
./basic/test-race-condition.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

./basic/test-duplicate.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

./basic/test-throughput-baseline.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

./basic/asic/test-order-sequence.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

./basic/test-crash-recovery.sh

echo ""
echo "╔════════════════════════════════════════════╗"
echo "║          BASELINE TEST SUMMARY             ║"
echo "╚════════════════════════════════════════════╝"
echo ""
echo "Issues Demonstrated:"
echo "✓ Race conditions from lack of partition key"
echo "✓ Duplicate processing from no idempotency"
echo "✓ Low throughput from no batching"
echo "✓ Out-of-order processing from random partitioning"
echo "✓ Data loss/duplicates from auto-commit"
echo ""
echo "Next: Run optimized version to see improvements!"
echo "      ./run-all-optimized-tests.sh"
