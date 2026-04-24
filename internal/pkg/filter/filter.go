package filter

import (
	"strings"
	"time"

	"bookcabin/internal/pkg/domain"
)

func Apply(flights []domain.Flight, f *domain.Filters) []domain.Flight {
	if f == nil {
		return flights
	}
	airlineSet := map[string]struct{}{}
	for _, a := range f.Airlines {
		airlineSet[strings.ToUpper(strings.TrimSpace(a))] = struct{}{}
	}

	out := flights[:0]
	for _, fl := range flights {
		if !matches(fl, f, airlineSet) {
			continue
		}
		out = append(out, fl)
	}
	result := make([]domain.Flight, len(out))
	copy(result, out)
	return result
}

func matches(fl domain.Flight, f *domain.Filters, airlineSet map[string]struct{}) bool {
	if f.MinPrice > 0 && fl.Price.Amount < f.MinPrice {
		return false
	}
	if f.MaxPrice > 0 && fl.Price.Amount > f.MaxPrice {
		return false
	}
	if f.MaxStops != nil && fl.Stops > *f.MaxStops {
		return false
	}
	if f.MaxDurationMins > 0 && fl.Duration.TotalMinutes > f.MaxDurationMins {
		return false
	}
	if len(airlineSet) > 0 {
		_, ok := airlineSet[strings.ToUpper(fl.Airline.Code)]
		if !ok {
			_, ok = airlineSet[strings.ToUpper(fl.Airline.Name)]
		}
		if !ok {
			return false
		}
	}
	if f.DepartureWindow != nil && !inHourWindow(fl.Departure.Timestamp, *f.DepartureWindow) {
		return false
	}
	if f.ArrivalWindow != nil && !inHourWindow(fl.Arrival.Timestamp, *f.ArrivalWindow) {
		return false
	}
	return true
}

func inHourWindow(ts int64, w domain.TimeWindow) bool {
	h := time.Unix(ts, 0).UTC().Hour()
	if w.FromHour <= w.ToHour {
		return h >= w.FromHour && h <= w.ToHour
	}
	return h >= w.FromHour || h <= w.ToHour
}
