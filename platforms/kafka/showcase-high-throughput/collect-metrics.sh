#!/bin/bash

echo "📊 Collecting Baseline Metrics..."

# Database stats
echo ""
echo "Database Stats:"
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    SELECT 
        COUNT(DISTINCT order_id) as unique_orders,
        COUNT(*) as total_processed,
        COUNT(*) - COUNT(DISTINCT order_id) as duplicates,
        SUM(quantity) as total_quantity
    FROM processed_orders;
"

# Product inventory
echo ""
echo "Product Inventory:"
docker exec inventory-postgres psql -U postgres -d inventory_db -c "
    SELECT 
        product_id,
        stock_quantity,
        reserved_quantity,
        available_quantity
    FROM products
    ORDER BY product_id;
"

# Kafka topic stats
echo ""
echo "Kafka Topic Stats:"
docker exec redpanda rpk topic describe orders
