// Package domain defines the core data types shared across all layers of the application.
// All provider-specific response shapes are normalized into these types before
// being processed by the aggregator, filter, ranker, and cache packages.
package domain

import (
	"errors"
	"fmt"
	"time"
)

// Airline holds the name and IATA code of a carrier.
type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Endpoint represents one end of a flight leg — either departure or arrival.
type Endpoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	DateTime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

// Duration holds a flight's total elapsed time in two forms:
// a raw minute count and a human-readable string (e.g. "2h 05m").
type Duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

// Price holds a flight's cost together with display helpers.
// BestValue is a 0–1 composite score set by [ranker.Rank]; it is zero until ranking runs.
type Price struct {
	Amount    int64   `json:"amount"`
	Currency  string  `json:"currency"`
	Formatted string  `json:"formatted,omitempty"`
	BestValue float64 `json:"best_value_score,omitempty"`
}

// Baggage describes the carry-on and checked baggage allowance as free-form strings
// because each provider expresses allowances differently.
type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

// Flight is the normalized representation of a single flight option returned
// by any provider. All provider-specific formats are converted to this struct
// before aggregation.
type Flight struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	Airline        Airline  `json:"airline"`
	FlightNumber   string   `json:"flight_number"`
	Departure      Endpoint `json:"departure"`
	Arrival        Endpoint `json:"arrival"`
	Duration       Duration `json:"duration"`
	Stops          int      `json:"stops"`
	Price          Price    `json:"price"`
	AvailableSeats int      `json:"available_seats"`
	CabinClass     string   `json:"cabin_class"`
	Aircraft       *string  `json:"aircraft"`
	Amenities      []string `json:"amenities"`
	Baggage        Baggage  `json:"baggage"`
}

// Validate reports whether the flight contains internally consistent data.
// It returns an error if timestamps are missing, arrival is not after departure,
// price is non-positive, duration is non-positive, or stops is negative.
// Providers drop flights that fail validation rather than surfacing bad data.
func (f *Flight) Validate() error {
	if f.Departure.Timestamp == 0 || f.Arrival.Timestamp == 0 {
		return errors.New("missing timestamps")
	}
	if f.Arrival.Timestamp <= f.Departure.Timestamp {
		return fmt.Errorf("flight %s: arrival not after departure", f.ID)
	}
	if f.Price.Amount <= 0 {
		return fmt.Errorf("flight %s: non-positive price", f.ID)
	}
	if f.Duration.TotalMinutes <= 0 {
		return fmt.Errorf("flight %s: non-positive duration", f.ID)
	}
	if f.Stops < 0 {
		return fmt.Errorf("flight %s: negative stops", f.ID)
	}
	return nil
}

// FormatDuration converts a total number of minutes into a human-readable
// string of the form "2h 05m". Negative values are treated as zero.
func FormatDuration(mins int) string {
	if mins < 0 {
		mins = 0
	}
	h := mins / 60
	m := mins % 60
	return fmt.Sprintf("%dh %02dm", h, m)
}

// DurationMinutes returns the elapsed time between dep and arr in whole minutes.
// The result is negative if arr is before dep.
func DurationMinutes(dep, arr time.Time) int {
	return int(arr.Sub(dep).Minutes())
}
