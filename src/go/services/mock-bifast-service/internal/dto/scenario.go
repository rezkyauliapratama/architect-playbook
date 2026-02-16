// src/go/services/mock-bifast-service/internal/dto/scenario.go
package dto

// TestScenario represents test scenario configuration untuk testing
type TestScenario struct {
	Name        string `json:"name"`        // Scenario name
	Description string `json:"description"` // Scenario description
	Trigger     string `json:"trigger"`     // Apa yang trigger scenario ini
	Behavior    string `json:"behavior"`    // Expected behavior dari scenario
}

// TestScenarios contains predefined test scenarios untuk mock service
var TestScenarios = []TestScenario{
	{
		Name:        "Success Scenario",
		Description: "Normal successful transaction",
		Trigger:     "Any valid transfer request",
		Behavior:    "Transaction completes successfully after 1-2 seconds",
	},
	{
		Name:        "Account Not Found",
		Description: "Destination account does not exist",
		Trigger:     "Account number ending with '0000'",
		Behavior:    "Returns error code 41 (Account not found)",
	},
	{
		Name:        "Insufficient Balance",
		Description: "Source account has insufficient balance",
		Trigger:     "Amount greater than 100,000,000",
		Behavior:    "Returns error code 42 (Insufficient balance)",
	},
	{
		Name:        "Timeout Simulation",
		Description: "Transaction timeout during processing",
		Trigger:     "Reference ID contains 'TIMEOUT'",
		Behavior:    "Transaction fails with timeout error after delay",
	},
	{
		Name:        "Duplicate Transaction",
		Description: "Duplicate idempotency key",
		Trigger:     "Same idempotency key used twice",
		Behavior:    "Returns error code 45 (Duplicate transaction)",
	},
}

// ScenarioResponse represents response yang contains list of test scenarios
type ScenarioResponse struct {
	Success   bool           `json:"success"`   // Status response
	Scenarios []TestScenario `json:"scenarios"` // List available scenarios
}
