package airport

import "time"

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

func Lookup(code string) Info {
	if info, ok := table[code]; ok {
		return info
	}
	return Info{Code: code, City: code, TZ: "UTC"}
}

func Location(code string) *time.Location {
	loc, err := time.LoadLocation(Lookup(code).TZ)
	if err != nil {
		return time.UTC
	}
	return loc
}
