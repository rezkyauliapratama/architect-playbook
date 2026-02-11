package dto

// StatisticsResponse represents transaction statistics
type StatisticsResponse struct {
	Success bool       `json:"success"`
	Data    Statistics `json:"data"`
}

// Statistics contains aggregated transaction data
type Statistics struct {
	TotalTransactions int    `json:"totalTransactions"`
	CompletedCount    int    `json:"completedCount"`
	FailedCount       int    `json:"failedCount"`
	PendingCount      int    `json:"pendingCount"`
	TotalAmount       string `json:"totalAmount"`
	TotalFee          string `json:"totalFee"`
}
