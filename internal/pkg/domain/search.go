package domain

import (
	"errors"
	"time"
)

type SortField string

const (
	SortPrice         SortField = "price"
	SortDuration      SortField = "duration"
	SortDepartureTime SortField = "departure_time"
	SortArrivalTime   SortField = "arrival_time"
	SortBestValue     SortField = "best_value"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type TimeWindow struct {
	FromHour int `json:"from_hour"`
	ToHour   int `json:"to_hour"`
}

type Filters struct {
	MinPrice        int64       `json:"min_price,omitempty"`
	MaxPrice        int64       `json:"max_price,omitempty"`
	MaxStops        *int        `json:"max_stops,omitempty"`
	Airlines        []string    `json:"airlines,omitempty"`
	MaxDurationMins int         `json:"max_duration_minutes,omitempty"`
	DepartureWindow *TimeWindow `json:"departure_window,omitempty"`
	ArrivalWindow   *TimeWindow `json:"arrival_window,omitempty"`
}

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

type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type Metadata struct {
	TotalResults       int   `json:"total_results"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}

type SearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       Metadata       `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}
