package filter

import (
	"sort"

	"bookcabin/internal/pkg/domain"
)

// Sort reorders flights in place according to by and order.
// If by is empty it defaults to [domain.SortPrice]; if order is empty it defaults to [domain.SortAsc].
func Sort(flights []domain.Flight, by domain.SortField, order domain.SortOrder) {
	if by == "" {
		by = domain.SortPrice
	}
	if order == "" {
		order = domain.SortAsc
	}
	less := comparator(by)
	sort.SliceStable(flights, func(i, j int) bool {
		if order == domain.SortDesc {
			return less(flights[j], flights[i])
		}
		return less(flights[i], flights[j])
	})
}

// comparator returns a less function for the given sort field.
func comparator(by domain.SortField) func(a, b domain.Flight) bool {
	switch by {
	case domain.SortDuration:
		return func(a, b domain.Flight) bool { return a.Duration.TotalMinutes < b.Duration.TotalMinutes }
	case domain.SortDepartureTime:
		return func(a, b domain.Flight) bool { return a.Departure.Timestamp < b.Departure.Timestamp }
	case domain.SortArrivalTime:
		return func(a, b domain.Flight) bool { return a.Arrival.Timestamp < b.Arrival.Timestamp }
	case domain.SortBestValue:
		// Higher score is better, so "less" means higher score.
		return func(a, b domain.Flight) bool { return a.Price.BestValue > b.Price.BestValue }
	default:
		return func(a, b domain.Flight) bool { return a.Price.Amount < b.Price.Amount }
	}
}
