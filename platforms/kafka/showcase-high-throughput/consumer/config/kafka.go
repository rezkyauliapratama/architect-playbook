package config

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ConsumerConfig returns production-grade Kafka consumer configuration
func ConsumerConfig(brokers, groupID, instanceID string) *kafka.ConfigMap {
	return &kafka.ConfigMap{
		// ===== CONNECTION =====
		"bootstrap.servers": brokers,
		"client.id":         "inventory-service-consumer",

		// ===== CONSUMER GROUP =====
		"group.id": groupID,

		// IMPORTANT: Static group membership (Kafka 2.3+)
		// Prevents unnecessary rebalancing on consumer restart
		"group.instance.id": instanceID,

		// ===== CONSISTENCY (Pillar 1) =====
		// Manual offset commit (commit only after successful processing)
		"enable.auto.commit": false,

		// Read only committed messages (for transactional producers)
		"isolation.level": "read_committed",

		// Where to start reading if no committed offset exists
		// Options: earliest, latest, error
		"auto.offset.reset": "earliest",

		// ===== THROUGHPUT (Pillar 2) =====
		// Fetch settings: accumulate data before returning to consumer
		"fetch.min.bytes":           1048576,  // Wait for 1MB
		"fetch.wait.max.ms":         100,      // Or wait max 100ms
		"max.partition.fetch.bytes": 10485760, // 10MB per partition

		// ===== FAULT TOLERANCE (Pillar 3) =====
		// Session timeout: Broker waits this long before removing consumer
		"session.timeout.ms": 30000, // 30 seconds

		// Heartbeat interval: Must be < session.timeout.ms / 3
		"heartbeat.interval.ms": 3000, // 3 seconds

		// Max poll interval: Max time between poll() calls
		// If exceeded, consumer is removed from group
		"max.poll.interval.ms": 300000, // 5 minutes

		// Partition assignment strategy
		// Options: range, roundrobin, cooperative-sticky
		"partition.assignment.strategy": "range",

		// ===== MONITORING & DEBUGGING =====
		"statistics.interval.ms": 5000,
		"log_level":              6,
	}
}
