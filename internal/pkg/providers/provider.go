package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"bookcabin/internal/pkg/domain"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error)
}

var ErrProviderUnavailable = errors.New("provider unavailable")

type URLs struct {
	Garuda  string
	Lion    string
	Batik   string
	AirAsia string
}

func All(u URLs) []Provider {
	return []Provider{
		&Garuda{BaseURL: u.Garuda},
		&Lion{BaseURL: u.Lion},
		&Batik{BaseURL: u.Batik},
		&AirAsia{BaseURL: u.AirAsia},
	}
}

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
