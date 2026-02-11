package dto

// TestScenario represents test scenario configuration
type TestScenario struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Trigger     string `json:"trigger"`  // What triggers this scenario
	Behavior    string `json:"behavior"` // Expected behavior
}

// Predefined test scenarios
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

// ScenarioResponse represents scenario list response
type ScenarioResponse struct {
	Success   bool           `json:"success"`
	Scenarios []TestScenario `json:"scenarios"`
}
