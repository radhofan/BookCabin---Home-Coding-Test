package providers

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"bookcabin/internal/pkg/airport"
	"bookcabin/internal/pkg/domain"
)

//go:embed testdata/lion.json
var lionRaw []byte

type Lion struct{}

func (Lion) Name() string { return "Lion Air" }

type lionResp struct {
	Success bool `json:"success"`
	Data    struct {
		Available []lionFlight `json:"available_flights"`
	} `json:"data"`
}

type lionAirport struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

type lionFlight struct {
	ID      string `json:"id"`
	Carrier struct {
		Name string `json:"name"`
		IATA string `json:"iata"`
	} `json:"carrier"`
	Route struct {
		From lionAirport `json:"from"`
		To   lionAirport `json:"to"`
	} `json:"route"`
	Schedule struct {
		Departure         string `json:"departure"`
		DepartureTimezone string `json:"departure_timezone"`
		Arrival           string `json:"arrival"`
		ArrivalTimezone   string `json:"arrival_timezone"`
	} `json:"schedule"`
	FlightTime int  `json:"flight_time"`
	IsDirect   bool `json:"is_direct"`
	StopCount  int  `json:"stop_count"`
	Layovers   []struct {
		Airport         string `json:"airport"`
		DurationMinutes int    `json:"duration_minutes"`
	} `json:"layovers,omitempty"`
	Pricing struct {
		Total    int64  `json:"total"`
		Currency string `json:"currency"`
		FareType string `json:"fare_type"`
	} `json:"pricing"`
	SeatsLeft int    `json:"seats_left"`
	PlaneType string `json:"plane_type"`
	Services  struct {
		WifiAvailable    bool `json:"wifi_available"`
		MealsIncluded    bool `json:"meals_included"`
		BaggageAllowance struct {
			Cabin string `json:"cabin"`
			Hold  string `json:"hold"`
		} `json:"baggage_allowance"`
	} `json:"services"`
}

func (l *Lion) Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	if err := simulate(ctx, 100*time.Millisecond, 200*time.Millisecond); err != nil {
		return nil, err
	}
	var resp lionResp
	if err := json.Unmarshal(lionRaw, &resp); err != nil {
		return nil, fmt.Errorf("lion decode: %w", err)
	}
	out := make([]domain.Flight, 0, len(resp.Data.Available))
	for _, f := range resp.Data.Available {
		nf, err := normalizeLion(f)
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

func normalizeLion(f lionFlight) (domain.Flight, error) {
	depLoc, err := time.LoadLocation(f.Schedule.DepartureTimezone)
	if err != nil {
		depLoc = airport.Location(f.Route.From.Code)
	}
	arrLoc, err := time.LoadLocation(f.Schedule.ArrivalTimezone)
	if err != nil {
		arrLoc = airport.Location(f.Route.To.Code)
	}
	depTime, err := time.ParseInLocation("2006-01-02T15:04:05", f.Schedule.Departure, depLoc)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("lion dep time: %w", err)
	}
	arrTime, err := time.ParseInLocation("2006-01-02T15:04:05", f.Schedule.Arrival, arrLoc)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("lion arr time: %w", err)
	}

	stops := f.StopCount
	if f.IsDirect {
		stops = 0
	}
	totalMins := f.FlightTime
	if computed := domain.DurationMinutes(depTime, arrTime); computed > 0 {
		totalMins = computed
	}

	var aircraft *string
	if f.PlaneType != "" {
		a := f.PlaneType
		aircraft = &a
	}

	amenities := make([]string, 0, 2)
	if f.Services.WifiAvailable {
		amenities = append(amenities, "wifi")
	}
	if f.Services.MealsIncluded {
		amenities = append(amenities, "meal")
	}

	nf := domain.Flight{
		ID:           f.ID + "_Lion",
		Provider:     "Lion Air",
		Airline:      domain.Airline{Name: f.Carrier.Name, Code: f.Carrier.IATA},
		FlightNumber: f.ID,
		Departure: domain.Endpoint{
			Airport:   f.Route.From.Code,
			City:      f.Route.From.City,
			DateTime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.Endpoint{
			Airport:   f.Route.To.Code,
			City:      f.Route.To.City,
			DateTime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration:       domain.Duration{TotalMinutes: totalMins, Formatted: domain.FormatDuration(totalMins)},
		Stops:          stops,
		Price:          domain.Price{Amount: f.Pricing.Total, Currency: f.Pricing.Currency},
		AvailableSeats: f.SeatsLeft,
		CabinClass:     normalizeCabin(f.Pricing.FareType),
		Aircraft:       aircraft,
		Amenities:      amenities,
		Baggage: domain.Baggage{
			CarryOn: f.Services.BaggageAllowance.Cabin,
			Checked: f.Services.BaggageAllowance.Hold,
		},
	}
	if err := nf.Validate(); err != nil {
		return domain.Flight{}, err
	}
	return nf, nil
}
