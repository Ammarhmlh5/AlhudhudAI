package testmode

import "fmt"

// TestPhoneNumbers maps test phone numbers to their expected results
var TestPhoneNumbers = map[string]string{
	// Success scenarios
	"+967777000001": "delivered",
	"+967777000011": "delivered",
	"+967777000021": "delivered",
	
	// Failure scenarios
	"+967777000002": "failed",
	"+967777000012": "failed",
	"+967777000022": "failed",
	
	// Pending scenarios
	"+967777000003": "pending",
	"+967777000013": "pending",
	"+967777000023": "pending",
	
	// Invalid number scenarios
	"+967777000004": "invalid_number",
	"+967777000014": "invalid_number",
	
	// Insufficient balance scenarios
	"+967777000005": "insufficient_balance",
	"+967777000015": "insufficient_balance",
	
	// Rate limited scenarios
	"+967777000006": "rate_limited",
	"+967777000016": "rate_limited",
	
	// International test numbers
	"+1555000001": "delivered",  // US
	"+44555000001": "delivered", // UK
	"+966555000001": "delivered", // Saudi Arabia
	"+971555000001": "delivered", // UAE
	"+20555000001": "delivered",  // Egypt
}

// IsTestPhoneNumber checks if phone number is a test number
func IsTestPhoneNumber(phoneNumber string) bool {
	_, exists := TestPhoneNumbers[phoneNumber]
	return exists
}

// GetTestResult returns the expected result for a test number
func GetTestResult(phoneNumber string) string {
	if result, exists := TestPhoneNumbers[phoneNumber]; exists {
		return result
	}
	// Default to delivered for unknown test numbers
	return "delivered"
}

// GetTestNumbersByResult returns all test numbers for a specific result
func GetTestNumbersByResult(result string) []string {
	var numbers []string
	for number, r := range TestPhoneNumbers {
		if r == result {
			numbers = append(numbers, number)
		}
	}
	return numbers
}

// TestNumberInfo provides information about test numbers
type TestNumberInfo struct {
	Number      string `json:"number"`
	Result      string `json:"result"`
	Description string `json:"description"`
}

// GetAllTestNumbers returns information about all test numbers
func GetAllTestNumbers() []TestNumberInfo {
	return []TestNumberInfo{
		{
			Number:      "+967777000001",
			Result:      "delivered",
			Description: "Message will be delivered successfully",
		},
		{
			Number:      "+967777000002",
			Result:      "failed",
			Description: "Message will fail to deliver",
		},
		{
			Number:      "+967777000003",
			Result:      "pending",
			Description: "Message will remain in pending status",
		},
		{
			Number:      "+967777000004",
			Result:      "invalid_number",
			Description: "Invalid phone number error",
		},
		{
			Number:      "+967777000005",
			Result:      "insufficient_balance",
			Description: "Insufficient balance error",
		},
		{
			Number:      "+967777000006",
			Result:      "rate_limited",
			Description: "Rate limit exceeded error",
		},
		{
			Number:      "+1555000001",
			Result:      "delivered",
			Description: "International test number (US)",
		},
		{
			Number:      "+44555000001",
			Result:      "delivered",
			Description: "International test number (UK)",
		},
		{
			Number:      "+966555000001",
			Result:      "delivered",
			Description: "International test number (Saudi Arabia)",
		},
	}
}

// ValidateTestNumber validates if a number is a valid test number
func ValidateTestNumber(phoneNumber string) (bool, string) {
	if !IsTestPhoneNumber(phoneNumber) {
		return false, "Not a test number. Use test numbers starting with +967777000xxx or +x555000xxx"
	}
	
	result := GetTestResult(phoneNumber)
	return true, fmt.Sprintf("Valid test number. Expected result: %s", result)
}

// GetTestNumbersDocumentation returns documentation for test numbers
func GetTestNumbersDocumentation() string {
	return `
Test Phone Numbers Documentation
================================

Use these test phone numbers to simulate different message delivery scenarios without sending real messages or deducting balance.

Success Scenarios:
- +967777000001, +967777000011, +967777000021: Message delivered successfully

Failure Scenarios:
- +967777000002, +967777000012, +967777000022: Message delivery failed

Pending Scenarios:
- +967777000003, +967777000013, +967777000023: Message remains pending

Error Scenarios:
- +967777000004, +967777000014: Invalid phone number
- +967777000005, +967777000015: Insufficient balance
- +967777000006, +967777000016: Rate limit exceeded

International Test Numbers:
- +1555000001: US (delivered)
- +44555000001: UK (delivered)
- +966555000001: Saudi Arabia (delivered)
- +971555000001: UAE (delivered)
- +20555000001: Egypt (delivered)

Notes:
- Test numbers do not send real messages
- No balance is deducted for test messages
- Test operations are logged separately
- Use test API keys (pk_test_xxx or sk_test_xxx) to enable test mode
`
}
