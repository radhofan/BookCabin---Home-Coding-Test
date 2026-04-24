package providers

import (
	"strings"

	"bookcabin/internal/pkg/domain"
)

func flightMatches(f domain.Flight, req domain.SearchRequest) bool {
	if !strings.EqualFold(f.Departure.Airport, req.Origin) {
		return false
	}
	if !strings.EqualFold(f.Arrival.Airport, req.Destination) {
		return false
	}
	if len(f.Departure.DateTime) >= 10 && f.Departure.DateTime[:10] != req.DepartureDate {
		return false
	}
	if req.CabinClass != "" && f.CabinClass != "" &&
		!strings.EqualFold(f.CabinClass, req.CabinClass) {
		return false
	}
	if req.Passengers > 0 && f.AvailableSeats > 0 && f.AvailableSeats < req.Passengers {
		return false
	}
	return true
}

func normalizeCabin(v string) string {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "Y", "ECONOMY", "COACH":
		return "economy"
	case "W", "PREMIUM_ECONOMY", "PREMIUM ECONOMY":
		return "premium_economy"
	case "C", "J", "BUSINESS":
		return "business"
	case "F", "FIRST":
		return "first"
	case "":
		return ""
	default:
		return strings.ToLower(v)
	}
}
