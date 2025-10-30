#!/bin/bash

# Delete topic
docker exec redpanda-0 rpk topic delete orders \
  --brokers redpanda-0:9092,redpanda-1:9092,redpanda-2:9092


# Create topic with production-grade configuration
echo "📝 Creating 'orders' topic..."
docker exec redpanda-0 rpk topic create orders \
  --partitions 6 \
  --replicas 3 \
  --config compression.type=lz4 \
  --config retention.ms=604800000 \
  --config min.insync.replicas=2 \
  --brokers redpanda-0:9092,redpanda-1:9092,redpanda-2:9092

# Describe topic
# docker exec redpanda-0 rpk topic describe orders