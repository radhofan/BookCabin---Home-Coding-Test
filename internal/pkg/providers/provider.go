package providers

import (
	"context"
	_ "embed"
	"errors"
	"math/rand"
	"time"

	"bookcabin/internal/pkg/domain"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error)
}

func simulate(ctx context.Context, min, max time.Duration) error {
	d := min + time.Duration(rand.Int63n(int64(max-min+1)))
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var ErrProviderUnavailable = errors.New("provider unavailable")

func All() []Provider {
	return []Provider{
		&Garuda{},
		&Lion{},
		&Batik{},
		&AirAsia{},
	}
}
