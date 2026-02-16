// src/go/services/mock-bifast-service/internal/dto/health.go
package dto

// HealthResponse represents response dari health check endpoint
type HealthResponse struct {
	Status      string            `json:"status"`           // Status service (healthy/unhealthy)
	Service     string            `json:"service"`          // Service name
	Version     string            `json:"version"`          // Service version
	Environment string            `json:"environment"`      // Environment (dev/staging/prod)
	Timestamp   string            `json:"timestamp"`        // Current timestamp ISO 8601
	Checks      map[string]string `json:"checks,omitempty"` // Health check results untuk dependencies
}

// HealthStatus represents individual health check status untuk dependency
type HealthStatus struct {
	Name   string `json:"name"`            // Nama dependency (database, redis, etc)
	Status string `json:"status"`          // Status: "up", "down", "degraded"
	Error  string `json:"error,omitempty"` // Error message jika ada
}
