package domain

import (
	"errors"
	"time"
)

// SortField names the field used to order search results.
type SortField string

const (
	SortPrice         SortField = "price"
	SortDuration      SortField = "duration"
	SortDepartureTime SortField = "departure_time"
	SortArrivalTime   SortField = "arrival_time"
	// SortBestValue orders results by the composite best-value score computed by [ranker.Rank].
	SortBestValue SortField = "best_value"
)

// SortOrder controls whether results are sorted ascending or descending.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// TimeWindow restricts flights to those departing or arriving within a range of hours.
// Hours are in 24-hour format (0–23). If ToHour is less than FromHour the window
// wraps midnight (e.g. FromHour=22, ToHour=2 matches 22:00–02:00).
type TimeWindow struct {
	FromHour int `json:"from_hour"`
	ToHour   int `json:"to_hour"`
}

// Filters narrows the flight list returned by a search. All fields are optional;
// a zero value means the corresponding filter is not applied.
type Filters struct {
	MinPrice        int64       `json:"min_price,omitempty"`
	MaxPrice        int64       `json:"max_price,omitempty"`
	MaxStops        *int        `json:"max_stops,omitempty"`
	Airlines        []string    `json:"airlines,omitempty"`
	MaxDurationMins int         `json:"max_duration_minutes,omitempty"`
	DepartureWindow *TimeWindow `json:"departure_window,omitempty"`
	ArrivalWindow   *TimeWindow `json:"arrival_window,omitempty"`
}

// SearchRequest is the body of a POST /search request.
// Origin, Destination, DepartureDate, Passengers, and CabinClass are required.
// Filters, SortBy, and SortOrder are optional and applied after aggregation.
// Set ReturnDate to request a round-trip; omit it or set it to nil for one-way.
type SearchRequest struct {
	Origin        string    `json:"origin"`
	Destination   string    `json:"destination"`
	DepartureDate string    `json:"departureDate"`
	ReturnDate    *string   `json:"returnDate,omitempty"`
	Passengers    int       `json:"passengers"`
	CabinClass    string    `json:"cabinClass"`
	Filters       *Filters  `json:"filters,omitempty"`
	SortBy        SortField `json:"sort_by,omitempty"`
	SortOrder     SortOrder `json:"sort_order,omitempty"`
}

// Validate reports whether the request contains all required fields with valid values.
// CabinClass defaults to "economy" when empty.
func (r *SearchRequest) Validate() error {
	if r.Origin == "" || r.Destination == "" {
		return errors.New("origin and destination required")
	}
	if r.Origin == r.Destination {
		return errors.New("origin and destination must differ")
	}
	if r.DepartureDate == "" {
		return errors.New("departureDate required")
	}
	if _, err := time.Parse("2006-01-02", r.DepartureDate); err != nil {
		return errors.New("departureDate must be YYYY-MM-DD")
	}
	if r.ReturnDate != nil && *r.ReturnDate != "" {
		if _, err := time.Parse("2006-01-02", *r.ReturnDate); err != nil {
			return errors.New("returnDate must be YYYY-MM-DD")
		}
	}
	if r.Passengers < 1 {
		return errors.New("passengers must be >= 1")
	}
	if r.CabinClass == "" {
		r.CabinClass = "economy"
	}
	return nil
}

// SearchCriteria echoes the core search parameters back in the response so callers
// can confirm what was searched without re-parsing their request.
type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

// Metadata contains diagnostic information about a completed search:
// how many providers were contacted, how many succeeded, total result count,
// elapsed time, and whether the response was served from cache.
type Metadata struct {
	TotalResults       int   `json:"total_results"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}

// SearchResponse is the result for a single flight leg.
// It is embedded in [service.TripResponse] which wraps both outbound and inbound legs.
type SearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       Metadata       `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}
