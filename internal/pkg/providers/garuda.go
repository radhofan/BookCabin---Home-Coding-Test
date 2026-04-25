package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bookcabin/internal/pkg/airport"
	"bookcabin/internal/pkg/domain"
)

// Garuda is the [Provider] adapter for Garuda Indonesia.
// BaseURL must be the base URL of the Garuda mock server (e.g. "http://127.0.0.1:PORT").
type Garuda struct{ BaseURL string }

func (Garuda) Name() string { return "Garuda Indonesia" }

type garudaResp struct {
	Status  string         `json:"status"`
	Flights []garudaFlight `json:"flights"`
}

type garudaEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type garudaSegment struct {
	FlightNumber    string         `json:"flight_number"`
	Departure       garudaEndpoint `json:"departure"`
	Arrival         garudaEndpoint `json:"arrival"`
	DurationMinutes int            `json:"duration_minutes"`
	LayoverMinutes  int            `json:"layover_minutes,omitempty"`
}

type garudaFlight struct {
	FlightID        string         `json:"flight_id"`
	Airline         string         `json:"airline"`
	AirlineCode     string         `json:"airline_code"`
	Departure       garudaEndpoint `json:"departure"`
	Arrival         garudaEndpoint `json:"arrival"`
	DurationMinutes int            `json:"duration_minutes"`
	Stops           int            `json:"stops"`
	Aircraft        string         `json:"aircraft"`
	Price           struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"price"`
	Segments       []garudaSegment `json:"segments,omitempty"`
	AvailableSeats int             `json:"available_seats"`
	FareClass      string          `json:"fare_class"`
	Baggage        struct {
		CarryOn int `json:"carry_on"`
		Checked int `json:"checked"`
	} `json:"baggage"`
	Amenities []string `json:"amenities"`
}

func (g *Garuda) Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	body, err := httpGet(ctx, g.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("garuda: %w", err)
	}
	var resp garudaResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("garuda decode: %w", err)
	}
	out := make([]domain.Flight, 0, len(resp.Flights))
	for _, f := range resp.Flights {
		nf, err := normalizeGaruda(f)
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

func normalizeGaruda(f garudaFlight) (domain.Flight, error) {
	dep := f.Departure
	arr := f.Arrival
	stops := f.Stops
	totalMins := f.DurationMinutes

	if len(f.Segments) > 1 {
		dep = f.Segments[0].Departure
		arr = f.Segments[len(f.Segments)-1].Arrival
		stops = len(f.Segments) - 1
	}

	depTime, err := time.Parse(time.RFC3339, dep.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("garuda dep time: %w", err)
	}
	arrTime, err := time.Parse(time.RFC3339, arr.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("garuda arr time: %w", err)
	}
	if computed := domain.DurationMinutes(depTime, arrTime); computed > 0 {
		totalMins = computed
	}

	var aircraft *string
	if f.Aircraft != "" {
		a := f.Aircraft
		aircraft = &a
	}

	depCity := dep.City
	if depCity == "" {
		depCity = airport.Lookup(dep.Airport).City
	}
	arrCity := arr.City
	if arrCity == "" {
		arrCity = airport.Lookup(arr.Airport).City
	}

	nf := domain.Flight{
		ID:           f.FlightID + "_Garuda",
		Provider:     "Garuda Indonesia",
		Airline:      domain.Airline{Name: f.Airline, Code: f.AirlineCode},
		FlightNumber: f.FlightID,
		Departure: domain.Endpoint{
			Airport:   dep.Airport,
			City:      depCity,
			DateTime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.Endpoint{
			Airport:   arr.Airport,
			City:      arrCity,
			DateTime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration:       domain.Duration{TotalMinutes: totalMins, Formatted: domain.FormatDuration(totalMins)},
		Stops:          stops,
		Price:          domain.Price{Amount: f.Price.Amount, Currency: f.Price.Currency},
		AvailableSeats: f.AvailableSeats,
		CabinClass:     f.FareClass,
		Aircraft:       aircraft,
		Amenities:      f.Amenities,
		Baggage: domain.Baggage{
			CarryOn: fmt.Sprintf("%d piece(s)", f.Baggage.CarryOn),
			Checked: fmt.Sprintf("%d piece(s)", f.Baggage.Checked),
		},
	}
	if err := nf.Validate(); err != nil {
		return domain.Flight{}, err
	}
	return nf, nil
}
