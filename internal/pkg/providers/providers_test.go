package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bookcabin/internal/pkg/domain"
)

func req() domain.SearchRequest {
	return domain.SearchRequest{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func jsonServer(t *testing.T, data []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGaruda_FiltersAndNormalizes(t *testing.T) {
	srv := jsonServer(t, mustReadFile(t, "testdata/garuda.json"))
	g := &Garuda{BaseURL: srv.URL}
	got, err := g.Fetch(context.Background(), req())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected flights")
	}
	for _, f := range got {
		if f.Departure.Airport != "CGK" {
			t.Errorf("%s: departure %s != CGK", f.ID, f.Departure.Airport)
		}
		if f.Arrival.Airport != "DPS" {
			t.Errorf("%s: arrival %s != DPS", f.ID, f.Arrival.Airport)
		}
	}
}

func TestGaruda_MultiSegmentRecomputesArrival(t *testing.T) {
	srv := jsonServer(t, mustReadFile(t, "testdata/garuda.json"))
	g := &Garuda{BaseURL: srv.URL}
	got, err := g.Fetch(context.Background(), req())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	var ga315 *domain.Flight
	for i := range got {
		if got[i].FlightNumber == "GA315" {
			ga315 = &got[i]
			break
		}
	}
	if ga315 == nil {
		t.Fatal("GA315 not found — segment expansion broken")
	}
	if ga315.Arrival.Airport != "DPS" {
		t.Errorf("GA315 arrival %s, want DPS", ga315.Arrival.Airport)
	}
	if ga315.Stops != 1 {
		t.Errorf("GA315 stops %d, want 1", ga315.Stops)
	}
	if ga315.Duration.TotalMinutes <= 0 {
		t.Errorf("GA315 duration not positive: %d", ga315.Duration.TotalMinutes)
	}
}

func TestLion_HandlesNamedTimezone(t *testing.T) {
	srv := jsonServer(t, mustReadFile(t, "testdata/lion.json"))
	l := &Lion{BaseURL: srv.URL}
	got, err := l.Fetch(context.Background(), req())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected flights")
	}
	for _, f := range got {
		if f.Departure.Timestamp == 0 || f.Arrival.Timestamp == 0 {
			t.Errorf("%s: zero timestamps", f.ID)
		}
		if f.Arrival.Timestamp <= f.Departure.Timestamp {
			t.Errorf("%s: arrival not after departure", f.ID)
		}
	}
}

func TestBatik_HandlesCompactOffset(t *testing.T) {
	srv := jsonServer(t, mustReadFile(t, "testdata/batik.json"))
	b := &Batik{BaseURL: srv.URL}
	got, err := b.Fetch(context.Background(), req())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected flights")
	}
	for _, f := range got {
		if f.Price.Amount <= 0 {
			t.Errorf("%s: price <= 0", f.ID)
		}
	}
}

func TestAirAsia_FailureRate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	a := &AirAsia{BaseURL: srv.URL}
	_, err := a.Fetch(context.Background(), req())
	if err == nil {
		t.Fatal("expected failure")
	}
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("err = %v, want ErrProviderUnavailable", err)
	}
}

func TestAirAsia_NeverFails(t *testing.T) {
	srv := jsonServer(t, mustReadFile(t, "testdata/airasia.json"))
	a := &AirAsia{BaseURL: srv.URL}
	got, err := a.Fetch(context.Background(), req())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected flights")
	}
}

func TestParseTravelTime(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"1h 45m", 105},
		{"3h 5m", 185},
		{"2h", 120},
		{"45m", 45},
	}
	for _, c := range cases {
		if got := parseTravelTime(c.in); got != c.want {
			t.Errorf("parseTravelTime(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestFlightMatches_RejectsWrongDate(t *testing.T) {
	f := domain.Flight{
		Departure:      domain.Endpoint{Airport: "CGK", DateTime: "2025-12-14T06:00:00+07:00"},
		Arrival:        domain.Endpoint{Airport: "DPS"},
		CabinClass:     "economy",
		AvailableSeats: 10,
	}
	if flightMatches(f, req()) {
		t.Error("expected reject on wrong date")
	}
}
