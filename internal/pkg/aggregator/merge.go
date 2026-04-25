package aggregator

import "bookcabin/internal/pkg/domain"

// Dedup collapses flights that represent the same physical service — identified
// by airline IATA code, flight number, and departure day — keeping only the
// cheapest option across providers. This satisfies the requirement to compare
// prices for the same flight across different sources.
func Dedup(flights []domain.Flight) []domain.Flight {
	type key struct {
		code   string
		number string
		tsDay  int64
	}
	best := map[key]int{}
	out := make([]domain.Flight, 0, len(flights))

	for _, f := range flights {
		k := key{
			code:   f.Airline.Code,
			number: f.FlightNumber,
			tsDay:  f.Departure.Timestamp / 86400,
		}
		if idx, ok := best[k]; ok {
			if f.Price.Amount < out[idx].Price.Amount {
				out[idx] = f
			}
			continue
		}
		best[k] = len(out)
		out = append(out, f)
	}
	return out
}
