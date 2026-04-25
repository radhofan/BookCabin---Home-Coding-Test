// Package providers contains one HTTP client adapter per airline.
// Each adapter calls its dedicated mock airline server, parses the
// provider-specific response shape, and normalizes the result into
// the shared [domain.Flight] type.
package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"bookcabin/internal/pkg/domain"
)

// Provider is the interface implemented by each airline adapter.
// Name returns a human-readable label used in logs and metrics.
// Fetch contacts the provider and returns normalized flights matching req.
type Provider interface {
	Name() string
	Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error)
}

// ErrProviderUnavailable is returned by [Provider.Fetch] when the airline
// server responds with HTTP 503. The aggregator treats this as a retryable error.
var ErrProviderUnavailable = errors.New("provider unavailable")

// URLs holds the base URLs of the four mock airline HTTP servers.
// Each field is the scheme+host+port of one server, e.g. "http://127.0.0.1:54321".
type URLs struct {
	Garuda  string
	Lion    string
	Batik   string
	AirAsia string
}

// All returns one [Provider] per airline, each configured to call its
// corresponding mock server at the given URLs.
func All(u URLs) []Provider {
	return []Provider{
		&Garuda{BaseURL: u.Garuda},
		&Lion{BaseURL: u.Lion},
		&Batik{BaseURL: u.Batik},
		&AirAsia{BaseURL: u.AirAsia},
	}
}

// httpGet performs a GET request to url+"/search" with the given context.
// It returns [ErrProviderUnavailable] on HTTP 503 and a generic error for
// any other non-200 status.
func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/search", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, ErrProviderUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
