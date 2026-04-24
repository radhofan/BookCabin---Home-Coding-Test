package providers

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bookcabin/internal/pkg/airport"
	"bookcabin/internal/pkg/domain"
)

//go:embed testdata/batik.json
var batikRaw []byte

type Batik struct{}

func (Batik) Name() string { return "Batik Air" }

type batikResp struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Results []batikFlight `json:"results"`
}

type batikFlight struct {
	FlightNumber      string `json:"flightNumber"`
	AirlineName       string `json:"airlineName"`
	AirlineIATA       string `json:"airlineIATA"`
	Origin            string `json:"origin"`
	Destination       string `json:"destination"`
	DepartureDateTime string `json:"departureDateTime"`
	ArrivalDateTime   string `json:"arrivalDateTime"`
	TravelTime        string `json:"travelTime"`
	NumberOfStops     int    `json:"numberOfStops"`
	Connections       []struct {
		StopAirport  string `json:"stopAirport"`
		StopDuration string `json:"stopDuration"`
	} `json:"connections,omitempty"`
	Fare struct {
		BasePrice    int64  `json:"basePrice"`
		Taxes        int64  `json:"taxes"`
		TotalPrice   int64  `json:"totalPrice"`
		CurrencyCode string `json:"currencyCode"`
		Class        string `json:"class"`
	} `json:"fare"`
	SeatsAvailable  int      `json:"seatsAvailable"`
	AircraftModel   string   `json:"aircraftModel"`
	BaggageInfo     string   `json:"baggageInfo"`
	OnboardServices []string `json:"onboardServices"`
}

func (b *Batik) Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	if err := simulate(ctx, 200*time.Millisecond, 400*time.Millisecond); err != nil {
		return nil, err
	}
	var resp batikResp
	if err := json.Unmarshal(batikRaw, &resp); err != nil {
		return nil, fmt.Errorf("batik decode: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("batik status %d: %s", resp.Code, resp.Message)
	}
	out := make([]domain.Flight, 0, len(resp.Results))
	for _, f := range resp.Results {
		nf, err := normalizeBatik(f)
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

var batikTimeLayouts = []string{
	"2006-01-02T15:04:05-0700",
	time.RFC3339,
}

func parseBatikTime(s string) (time.Time, error) {
	for _, layout := range batikTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time %q", s)
}

var travelTimeRe = regexp.MustCompile(`(?i)(?:(\d+)\s*h)?\s*(?:(\d+)\s*m)?`)

func parseTravelTime(s string) int {
	m := travelTimeRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	return h*60 + min
}

func normalizeBatik(f batikFlight) (domain.Flight, error) {
	depTime, err := parseBatikTime(f.DepartureDateTime)
	if err != nil {
		return domain.Flight{}, err
	}
	arrTime, err := parseBatikTime(f.ArrivalDateTime)
	if err != nil {
		return domain.Flight{}, err
	}
	totalMins := parseTravelTime(f.TravelTime)
	if computed := domain.DurationMinutes(depTime, arrTime); computed > 0 {
		totalMins = computed
	}

	var aircraft *string
	if f.AircraftModel != "" {
		a := f.AircraftModel
		aircraft = &a
	}

	carryOn, checked := splitBaggage(f.BaggageInfo)

	nf := domain.Flight{
		ID:           f.FlightNumber + "_Batik",
		Provider:     "Batik Air",
		Airline:      domain.Airline{Name: f.AirlineName, Code: f.AirlineIATA},
		FlightNumber: f.FlightNumber,
		Departure: domain.Endpoint{
			Airport:   f.Origin,
			City:      airport.Lookup(f.Origin).City,
			DateTime:  depTime.Format(time.RFC3339),
			Timestamp: depTime.Unix(),
		},
		Arrival: domain.Endpoint{
			Airport:   f.Destination,
			City:      airport.Lookup(f.Destination).City,
			DateTime:  arrTime.Format(time.RFC3339),
			Timestamp: arrTime.Unix(),
		},
		Duration:       domain.Duration{TotalMinutes: totalMins, Formatted: domain.FormatDuration(totalMins)},
		Stops:          f.NumberOfStops,
		Price:          domain.Price{Amount: f.Fare.TotalPrice, Currency: f.Fare.CurrencyCode},
		AvailableSeats: f.SeatsAvailable,
		CabinClass:     normalizeCabin(f.Fare.Class),
		Aircraft:       aircraft,
		Amenities:      f.OnboardServices,
		Baggage:        domain.Baggage{CarryOn: carryOn, Checked: checked},
	}
	if err := nf.Validate(); err != nil {
		return domain.Flight{}, err
	}
	return nf, nil
}

func splitBaggage(s string) (string, string) {
	parts := strings.Split(s, ",")
	if len(parts) < 2 {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
