package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"bookcabin/internal/pkg/domain"
	"bookcabin/internal/pkg/providers"
)

type fakeProvider struct {
	name    string
	flights []domain.Flight
	err     error
	calls   int
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Fetch(_ context.Context, _ domain.SearchRequest) ([]domain.Flight, error) {
	f.calls++
	return f.flights, f.err
}

func TestAggregate_CollectsAll(t *testing.T) {
	ok := &fakeProvider{name: "OK", flights: []domain.Flight{{ID: "1"}}}
	bad := &fakeProvider{name: "BAD", err: errors.New("boom")}
	a := New(DefaultConfig(), []providers.Provider{ok, bad})

	res := a.Aggregate(context.Background(), domain.SearchRequest{})
	if len(res) != 2 {
		t.Fatalf("results len = %d", len(res))
	}
	var okOK, badOK bool
	for _, r := range res {
		if r.Name == "OK" && r.Err == nil && len(r.Flights) == 1 {
			okOK = true
		}
		if r.Name == "BAD" && r.Err != nil {
			badOK = true
		}
	}
	if !okOK || !badOK {
		t.Errorf("unexpected results: %+v", res)
	}
}

func TestAggregate_RetriesOnError(t *testing.T) {
	p := &fakeProvider{name: "flaky", err: errors.New("fail")}
	cfg := DefaultConfig()
	cfg.MaxRetries = 2
	cfg.BackoffBase = time.Millisecond
	cfg.BackoffMax = 2 * time.Millisecond
	a := New(cfg, []providers.Provider{p})

	_ = a.Aggregate(context.Background(), domain.SearchRequest{})
	if p.calls != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", p.calls)
	}
}

func TestDedup_KeepsCheapest(t *testing.T) {
	flights := []domain.Flight{
		{FlightNumber: "GA400", Airline: domain.Airline{Code: "GA"}, Departure: domain.Endpoint{Timestamp: 1734246000}, Price: domain.Price{Amount: 1200000}},
		{FlightNumber: "GA400", Airline: domain.Airline{Code: "GA"}, Departure: domain.Endpoint{Timestamp: 1734246000}, Price: domain.Price{Amount: 950000}},
		{FlightNumber: "JT740", Airline: domain.Airline{Code: "JT"}, Departure: domain.Endpoint{Timestamp: 1734248000}, Price: domain.Price{Amount: 800000}},
	}
	got := Dedup(flights)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	for _, f := range got {
		if f.FlightNumber == "GA400" && f.Price.Amount != 950000 {
			t.Errorf("GA400 kept %d, want 950000", f.Price.Amount)
		}
	}
}
