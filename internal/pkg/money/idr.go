// Package money formats currency amounts for display.
package money

import (
	"strconv"
	"strings"
)

// FormatIDR formats amount as an Indonesian Rupiah string with dot thousands
// separators, e.g. 1250000 → "Rp 1.250.000". Negative amounts are prefixed with "-".
func FormatIDR(amount int64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := strconv.FormatInt(amount, 10)
	var b strings.Builder
	b.WriteString("Rp ")
	n := len(s)
	first := n % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(s[:first])
	for i := first; i < n; i += 3 {
		b.WriteByte('.')
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// Format formats amount in the given currency. "IDR" uses [FormatIDR].
// All other currencies are formatted as "<CURRENCY> <amount>".
func Format(amount int64, currency string) string {
	switch strings.ToUpper(currency) {
	case "IDR":
		return FormatIDR(amount)
	default:
		return currency + " " + strconv.FormatInt(amount, 10)
	}
}
