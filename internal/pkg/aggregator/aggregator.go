package aggregator

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"bookcabin/internal/pkg/domain"
	"bookcabin/internal/pkg/providers"
)

type ProviderResult struct {
	Name    string
	Flights []domain.Flight
	Err     error
	Latency time.Duration
}

type Config struct {
	PerProviderTimeout time.Duration
	MaxRetries         int
	BackoffBase        time.Duration
	BackoffMax         time.Duration
	RateLimitCapacity  int
	RateLimitRPS       float64
}

func DefaultConfig() Config {
	return Config{
		PerProviderTimeout: 2 * time.Second,
		MaxRetries:         2,
		BackoffBase:        100 * time.Millisecond,
		BackoffMax:         1 * time.Second,
		RateLimitCapacity:  10,
		RateLimitRPS:       20,
	}
}

type Aggregator struct {
	cfg       Config
	providers []providers.Provider
	limiters  map[string]*Limiter
}

func New(cfg Config, ps []providers.Provider) *Aggregator {
	limiters := make(map[string]*Limiter, len(ps))
	for _, p := range ps {
		limiters[p.Name()] = NewLimiter(cfg.RateLimitCapacity, cfg.RateLimitRPS)
	}
	return &Aggregator{cfg: cfg, providers: ps, limiters: limiters}
}

func (a *Aggregator) Aggregate(ctx context.Context, req domain.SearchRequest) []ProviderResult {
	results := make([]ProviderResult, len(a.providers))
	var wg sync.WaitGroup
	for i, p := range a.providers {
		wg.Add(1)
		go func(i int, p providers.Provider) {
			defer wg.Done()
			start := time.Now()
			flights, err := a.callWithRetry(ctx, p, req)
			results[i] = ProviderResult{
				Name:    p.Name(),
				Flights: flights,
				Err:     err,
				Latency: time.Since(start),
			}
			if err != nil {
				log.Printf("provider=%s err=%v latency=%s", p.Name(), err, time.Since(start))
			}
		}(i, p)
	}
	wg.Wait()
	return results
}

func (a *Aggregator) callWithRetry(ctx context.Context, p providers.Provider, req domain.SearchRequest) ([]domain.Flight, error) {
	pCtx, cancel := context.WithTimeout(ctx, a.cfg.PerProviderTimeout)
	defer cancel()

	limiter := a.limiters[p.Name()]
	var lastErr error
	for attempt := 0; attempt <= a.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(a.cfg.BackoffBase, a.cfg.BackoffMax, attempt)
			t := time.NewTimer(delay)
			select {
			case <-pCtx.Done():
				t.Stop()
				return nil, pCtx.Err()
			case <-t.C:
			}
		}
		if limiter != nil {
			if err := limiter.Wait(pCtx); err != nil {
				return nil, err
			}
		}
		flights, err := p.Fetch(pCtx, req)
		if err == nil {
			return flights, nil
		}
		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
	}
	return nil, lastErr
}

func backoff(base, max time.Duration, attempt int) time.Duration {
	d := base << (attempt - 1)
	if d <= 0 || d > max {
		d = max
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}
