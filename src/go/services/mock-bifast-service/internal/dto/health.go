package dto

// HealthResponse represents health check response
type HealthResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Timestamp   string            `json:"timestamp"`
	Checks      map[string]string `json:"checks,omitempty"`
}

// HealthStatus represents individual health check status
type HealthStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "up", "down", "degraded"
	Error  string `json:"error,omitempty"`
}
