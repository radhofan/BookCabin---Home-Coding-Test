package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bookcabin/internal/pkg/airport"
	"bookcabin/internal/pkg/domain"
)

// AirAsia is the [Provider] adapter for AirAsia.
// BaseURL must be the base URL of the AirAsia mock server (e.g. "http://127.0.0.1:PORT").
type AirAsia struct{ BaseURL string }

func (AirAsia) Name() string { return "AirAsia" }

type airasiaResp struct {
	Status  string          `json:"status"`
	Flights []airasiaFlight `json:"flights"`
}

type airasiaFlight struct {
	FlightCode    string  `json:"flight_code"`
	Airline       string  `json:"airline"`
	FromAirport   string  `json:"from_airport"`
	ToAirport     string  `json:"to_airport"`
	DepartTime    string  `json:"depart_time"`
	ArriveTime    string  `json:"arrive_time"`
	DurationHours float64 `json:"duration_hours"`
	DirectFlight  bool    `json:"direct_flight"`
	Stops         []struct {
		Airport         string `json:"airport"`
		WaitTimeMinutes int    `json:"wait_time_minutes"`
	} `json:"stops,omitempty"`
	PriceIDR    int64  `json:"price_idr"`
	Seats       int    `json:"seats"`
	CabinClass  string `json:"cabin_class"`
	BaggageNote string `json:"baggage_note"`
}

func (a *AirAsia) Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	body, err := httpGet(ctx, a.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("airasia: %w", err)
	}
	var resp airasiaResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("airasia decode: %w", err)
	}
	out := make([]domain.Flight, 0, len(resp.Flights))
	for _, f := range resp.Flights {
		nf, err := normalizeAirAsia(f)
		if err != nil {
			continue
		}
		if !flightMatches(nf, req) {
			continue
		}
		out = append(out, nf)
	}
	return out, nil
}

func normalizeAirAsia(f airasiaFlight) (domain.Flight, error) {
	depTime, err := time.Parse(time.RFC3339, f.DepartTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("airasia dep: %w", err)
	}
	arrTime, err := time.Parse(time.RFC3339, f.ArriveTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("airasia arr: %w", err)
	}
	stops := 0
	if !f.DirectFlight {
		if len(f.Stops) > 0 {
			stops = len(f.Stops)
		} else {
			stops = 1
		}
	}
	totalMins := int(f.DurationHours * 60)
	if computed := domain.DurationMinutes(depTime, arrTime); computed > 0 {
		totalMins = computed
	}

	nf := domain.Flight{
		ID:           f.FlightCode + "_AirAsia",
		Provider:     "AirAsia",
		Airline:      domain.Airline{Name: f.Airline, Code: airlineCodeFromFlight(f.FlightCode)},
		FlightNumber: f.FlightCode,
		Departure: domain.Endpoint{
			Airport:   f.FromAirport,
			City:      airport.Lookup(f.FromAirport).City,
			DateTime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.Endpoint{
			Airport:   f.ToAirport,
			City:      airport.Lookup(f.ToAirport).City,
			DateTime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration:       domain.Duration{TotalMinutes: totalMins, Formatted: domain.FormatDuration(totalMins)},
		Stops:          stops,
		Price:          domain.Price{Amount: f.PriceIDR, Currency: "IDR"},
		AvailableSeats: f.Seats,
		CabinClass:     normalizeCabin(f.CabinClass),
		Aircraft:       nil,
		Amenities:      []string{},
		Baggage: domain.Baggage{
			CarryOn: "Cabin baggage only",
			Checked: "Additional fee",
		},
	}
	if err := nf.Validate(); err != nil {
		return domain.Flight{}, err
	}
	return nf, nil
}

func airlineCodeFromFlight(s string) string {
	for i, r := range s {
		if r >= '0' && r <= '9' {
			return s[:i]
		}
	}
	return s
}
