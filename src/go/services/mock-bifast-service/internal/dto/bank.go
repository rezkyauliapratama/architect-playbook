// src/go/services/mock-bifast-service/internal/dto/bank.go
package dto

// BankInfo represents informasi bank participant BI-FAST
type BankInfo struct {
	BankCode  string `json:"bankCode"`            // Bank code 8 karakter
	BankName  string `json:"bankName"`            // Nama bank lengkap
	SwiftCode string `json:"swiftCode,omitempty"` // SWIFT/BIC code (optional)
}

// SupportedBanksResponse represents list of supported banks
type SupportedBanksResponse struct {
	Success bool       `json:"success"` // Status response
	Data    []BankInfo `json:"data"`    // List supported banks
}

// SupportedBanks contains list of major Indonesian banks (BI-FAST participants)
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

// GetBankName returns bank name berdasarkan bank code
func GetBankName(bankCode string) string {
	for _, bank := range SupportedBanks {
		if bank.BankCode == bankCode {
			return bank.BankName
		}
	}
	return "Unknown Bank" // Default jika tidak ditemukan
}

// IsSupportedBank checks apakah bank code didukung oleh mock service
func IsSupportedBank(bankCode string) bool {
	for _, bank := range SupportedBanks {
		if bank.BankCode == bankCode {
			return true
		}
	}
	return false
}
