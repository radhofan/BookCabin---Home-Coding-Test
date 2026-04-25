// Package airport maps Indonesian airport IATA codes to city names and
// IANA timezone identifiers. It is used by provider normalizers to fill in
// city names and resolve named timezones when a provider omits them.
package airport

import "time"

// Info holds static metadata for a single airport.
type Info struct {
	Code string
	City string
	TZ   string
}

var table = map[string]Info{
	"CGK": {"CGK", "Jakarta", "Asia/Jakarta"},
	"DPS": {"DPS", "Denpasar", "Asia/Makassar"},
	"SUB": {"SUB", "Surabaya", "Asia/Jakarta"},
	"SOC": {"SOC", "Solo", "Asia/Jakarta"},
	"UPG": {"UPG", "Makassar", "Asia/Makassar"},
	"DJJ": {"DJJ", "Jayapura", "Asia/Jayapura"},
	"JOG": {"JOG", "Yogyakarta", "Asia/Jakarta"},
	"KNO": {"KNO", "Medan", "Asia/Jakarta"},
	"BPN": {"BPN", "Balikpapan", "Asia/Makassar"},
}

// Lookup returns the [Info] for the given IATA code.
// If the code is not in the table, it returns a fallback Info where City equals
// the code itself and TZ is "UTC".
func Lookup(code string) Info {
	if info, ok := table[code]; ok {
		return info
	}
	return Info{Code: code, City: code, TZ: "UTC"}
}

// Location returns the [time.Location] for the given IATA code.
// It falls back to [time.UTC] if the timezone cannot be loaded.
func Location(code string) *time.Location {
	loc, err := time.LoadLocation(Lookup(code).TZ)
	if err != nil {
		return time.UTC
	}
	return loc
}
