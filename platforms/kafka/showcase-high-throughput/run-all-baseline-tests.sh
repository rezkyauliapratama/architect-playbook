#!/bin/bash

echo "╔════════════════════════════════════════════╗"
echo "║  BASELINE (BEFORE) - COMPLETE TEST SUITE  ║"
echo "╚════════════════════════════════════════════╝"
echo ""

# Make all test scripts executable
chmod +x tests/test-*.sh

# Run all tests
sh create-topic.sh
echo ""
sh tests/test-race-condition.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

sh create-topic.sh
echo ""
sh tests/test-duplicate.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

sh create-topic.sh
echo ""
sh tests/test-throughput.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

sh create-topic.sh
echo ""
sh tests/test-order-sequence.sh
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

sh create-topic.sh
echo ""
sh tests/test-crash-recovery.sh

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
echo "      sh run-all-optimized-tests.sh"
