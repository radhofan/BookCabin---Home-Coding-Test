package money

import "testing"

func TestFormatIDR(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "Rp 0"},
		{500, "Rp 500"},
		{1000, "Rp 1.000"},
		{1250000, "Rp 1.250.000"},
		{123456789, "Rp 123.456.789"},
		{-50000, "-Rp 50.000"},
	}
	for _, c := range cases {
		if got := FormatIDR(c.in); got != c.want {
			t.Errorf("FormatIDR(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
