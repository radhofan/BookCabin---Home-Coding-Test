package filter

import (
	"testing"

	"bookcabin/internal/pkg/domain"
)

func sample() []domain.Flight {
	return []domain.Flight{
		{ID: "A", Price: domain.Price{Amount: 500000}, Stops: 0, Duration: domain.Duration{TotalMinutes: 100}, Airline: domain.Airline{Code: "GA"}, Departure: domain.Endpoint{Timestamp: 1734246000}, Arrival: domain.Endpoint{Timestamp: 1734252000}},
		{ID: "B", Price: domain.Price{Amount: 1200000}, Stops: 1, Duration: domain.Duration{TotalMinutes: 200}, Airline: domain.Airline{Code: "QZ"}, Departure: domain.Endpoint{Timestamp: 1734260000}, Arrival: domain.Endpoint{Timestamp: 1734272000}},
		{ID: "C", Price: domain.Price{Amount: 800000}, Stops: 0, Duration: domain.Duration{TotalMinutes: 110}, Airline: domain.Airline{Code: "JT"}, Departure: domain.Endpoint{Timestamp: 1734270000}, Arrival: domain.Endpoint{Timestamp: 1734276600}},
	}
}

func TestFilter_Price(t *testing.T) {
	got := Apply(sample(), &domain.Filters{MaxPrice: 900000})
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
}

func TestFilter_Airline(t *testing.T) {
	got := Apply(sample(), &domain.Filters{Airlines: []string{"qz", "jt"}})
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
}

func TestFilter_MaxStops(t *testing.T) {
	zero := 0
	got := Apply(sample(), &domain.Filters{MaxStops: &zero})
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
}

func TestSort_PriceAsc(t *testing.T) {
	fs := sample()
	Sort(fs, domain.SortPrice, domain.SortAsc)
	if fs[0].ID != "A" || fs[2].ID != "B" {
		t.Errorf("sort wrong: %v", []string{fs[0].ID, fs[1].ID, fs[2].ID})
	}
}

func TestSort_DurationDesc(t *testing.T) {
	fs := sample()
	Sort(fs, domain.SortDuration, domain.SortDesc)
	if fs[0].ID != "B" {
		t.Errorf("want B first, got %s", fs[0].ID)
	}
}
