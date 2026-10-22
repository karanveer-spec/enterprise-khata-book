package models

import (
	"fmt"
	"strings"
)

// PaiseToRupees converts integer paise to float64 rupees.
func PaiseToRupees(paise int64) float64 {
	return float64(paise) / 100.0
}

// RupeesToPaise converts float64 rupees to integer paise without floating point inaccuracy.
func RupeesToPaise(rupees float64) int64 {
	if rupees < 0 {
		return int64((rupees * 100.0) - 0.5)
	}
	return int64((rupees * 100.0) + 0.5)
}

// FormatIndianRupees formats integer paise as an Indian currency string: e.g. "₹ 1,50,000.00".
func FormatIndianRupees(paise int64) string {
	isNegative := paise < 0
	if isNegative {
		paise = -paise
	}

	rupeesPart := paise / 100
	paisePart := paise % 100

	rStr := fmt.Sprintf("%d", rupeesPart)
	n := len(rStr)

	var formattedRupees string
	if n <= 3 {
		formattedRupees = rStr
	} else {
		// Last 3 digits
		last3 := rStr[n-3:]
		rest := rStr[:n-3]

		// Group rest by 2 digits from right to left
		var parts []string
		for len(rest) > 2 {
			parts = append([]string{rest[len(rest)-2:]}, parts...)
			rest = rest[:len(rest)-2]
		}
		if len(rest) > 0 {
			parts = append([]string{rest}, parts...)
		}
		formattedRupees = strings.Join(parts, ",") + "," + last3
	}

	res := fmt.Sprintf("₹ %s.%02d", formattedRupees, paisePart)
	if isNegative {
		return "-" + res
	}
	return res
}
