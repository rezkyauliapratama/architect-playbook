// src/go/services/mock-bifast-service/internal/dto/statistic.go
package dto

// StatisticsResponse represents transaction statistics response
type StatisticsResponse struct {
	Success           bool    `json:"success"`
	TotalTransactions int     `json:"totalTransactions"`
	SuccessCount      int     `json:"successCount"`   // Alias for CompletedCount
	CompletedCount    int     `json:"completedCount"` // Kept for backward compatibility
	FailedCount       int     `json:"failedCount"`
	PendingCount      int     `json:"pendingCount"`
	ProcessingCount   int     `json:"processingCount"` // NEW: For transactions in processing state
	TotalAmount       string  `json:"totalAmount"`
	TotalFee          string  `json:"totalFee"`
	SuccessRate       float64 `json:"successRate"`
	Timestamp         string  `json:"timestamp"`
}

// Statistics contains aggregated transaction data (legacy - for backward compatibility)
type Statistics struct {
	TotalTransactions int    `json:"totalTransactions"` // Total transactions
	CompletedCount    int    `json:"completedCount"`    // Completed transactions
	FailedCount       int    `json:"failedCount"`       // Failed transactions
	PendingCount      int    `json:"pendingCount"`      // Pending transactions
	TotalAmount       string `json:"totalAmount"`       // Total amount
	TotalFee          string `json:"totalFee"`          // Total fee
}
