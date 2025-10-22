package config

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ProducerConfig returns production-grade Kafka producer configuration
func ProducerConfig(brokers string) *kafka.ConfigMap {
	return &kafka.ConfigMap{
		// ===== CONNECTION =====
		"bootstrap.servers": brokers,
		"client.id":         "order-service-producer",

		// ===== CONSISTENCY (Pillar 1) =====
		// Idempotent producer: prevents duplicate messages
		"enable.idempotence": true,

		// Acknowledgment level: wait for all in-sync replicas
		// Options: 0 (none), 1 (leader), all/-1 (all replicas)
		"acks": "all",

		// Max number of unacknowledged requests per connection
		// With idempotence: can be up to 5 (maintains order)
		"max.in.flight.requests.per.connection": 5,

		// Retry configuration
		"retries":             2147483647, // Max retries (effectively infinite)
		"retry.backoff.ms":    100,        // Wait 100ms between retries
		"request.timeout.ms":  30000,      // 30s timeout for broker response
		"delivery.timeout.ms": 120000,     // 2min total timeout for delivery

		// ===== THROUGHPUT (Pillar 2) =====
		// Batching: accumulate messages before sending
		"batch.size": 100000, // 100KB batches
		"linger.ms":  10,     // Wait max 10ms for batching

		// Compression: lz4 = best balance (speed + ratio)
		// Options: none, gzip, snappy, lz4, zstd
		"compression.type": "lz4",

		// ===== FAULT TOLERANCE (Pillar 3) =====
		// Socket settings
		"socket.keepalive.enable": true,
		"socket.timeout.ms":       60000, // 60s socket timeout

		// Metadata refresh
		"metadata.max.age.ms": 180000, // Refresh metadata every 3 min

		// ===== MONITORING & DEBUGGING =====
		// Enable statistics (every 5 seconds)
		"statistics.interval.ms": 5000,

		// Log level: 7=debug, 6=info, 5=notice, 4=warning, 3=error
		"log_level": 6,
	}
}
