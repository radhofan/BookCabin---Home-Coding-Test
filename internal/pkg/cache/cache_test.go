package cache

import (
	"testing"
	"time"

	"bookcabin/internal/pkg/domain"
)

func TestCache_GetSet(t *testing.T) {
	c := New(100 * time.Millisecond)
	resp := domain.SearchResponse{Flights: []domain.Flight{{ID: "x"}}}
	c.Set("k", resp)

	got, ok := c.Get("k")
	if !ok || len(got.Flights) != 1 {
		t.Fatalf("miss: ok=%v, flights=%d", ok, len(got.Flights))
	}

	time.Sleep(150 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Error("entry should have expired")
	}
}

func TestKey_StableAcrossCase(t *testing.T) {
	a := Key(domain.SearchRequest{Origin: "cgk", Destination: "DPS", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "Economy"})
	b := Key(domain.SearchRequest{Origin: "CGK", Destination: "dps", DepartureDate: "2025-12-15", Passengers: 1, CabinClass: "economy"})
	if a != b {
		t.Errorf("keys differ:\n a=%s\n b=%s", a, b)
	}
}
