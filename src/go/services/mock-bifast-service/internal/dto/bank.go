package dto

// BankInfo represents bank information
type BankInfo struct {
	BankCode  string `json:"bankCode"`
	BankName  string `json:"bankName"`
	SwiftCode string `json:"swiftCode,omitempty"`
}

// SupportedBanksResponse represents list of supported banks
type SupportedBanksResponse struct {
	Success bool       `json:"success"`
	Data    []BankInfo `json:"data"`
}

// Bank codes for major Indonesian banks (BI-FAST participants)
var SupportedBanks = []BankInfo{
	{BankCode: "CENAIDJA", BankName: "Bank Central Asia", SwiftCode: "CENAIDJA"},
	{BankCode: "BDINIDJA", BankName: "Bank Danamon Indonesia", SwiftCode: "BDINIDJA"},
	{BankCode: "BMRIIDJA", BankName: "Bank Mandiri", SwiftCode: "BMRIIDJA"},
	{BankCode: "BNIAIDJA", BankName: "Bank Negara Indonesia", SwiftCode: "BNIAIDJA"},
	{BankCode: "SNIAIDJA", BankName: "Bank Sinarmas", SwiftCode: "SNIAIDJA"},
	{BankCode: "BBRIIDJA", BankName: "Bank Rakyat Indonesia", SwiftCode: "BBRIIDJA"},
	{BankCode: "BRINIDJA", BankName: "Bank Rakyat Indonesia", SwiftCode: "BRINIDJA"},
	{BankCode: "PERMIDJA", BankName: "Bank Permata", SwiftCode: "PERMIDJA"},
	{BankCode: "CITIIDJA", BankName: "Citibank Indonesia", SwiftCode: "CITIIDJA"},
	{BankCode: "DEUTIDJA", BankName: "Deutsche Bank Indonesia", SwiftCode: "DEUTIDJA"},
}

// GetBankName returns bank name by code
func GetBankName(bankCode string) string {
	for _, bank := range SupportedBanks {
		if bank.BankCode == bankCode {
			return bank.BankName
		}
	}
	return "Unknown Bank"
}

// IsSupportedBank checks if bank code is supported
func IsSupportedBank(bankCode string) bool {
	for _, bank := range SupportedBanks {
		if bank.BankCode == bankCode {
			return true
		}
	}
	return false
}
