package domain

import (
	"errors"
	"fmt"
	"time"
)

type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type Endpoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	DateTime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

type Duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Price struct {
	Amount    int64   `json:"amount"`
	Currency  string  `json:"currency"`
	Formatted string  `json:"formatted,omitempty"`
	BestValue float64 `json:"best_value_score,omitempty"`
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

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

func FormatDuration(mins int) string {
	if mins < 0 {
		mins = 0
	}
	h := mins / 60
	m := mins % 60
	return fmt.Sprintf("%dh %02dm", h, m)
}

func DurationMinutes(dep, arr time.Time) int {
	return int(arr.Sub(dep).Minutes())
}
