package money

import (
	"strconv"
	"strings"
)

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

func Format(amount int64, currency string) string {
	switch strings.ToUpper(currency) {
	case "IDR":
		return FormatIDR(amount)
	default:
		return currency + " " + strconv.FormatInt(amount, 10)
	}
}
