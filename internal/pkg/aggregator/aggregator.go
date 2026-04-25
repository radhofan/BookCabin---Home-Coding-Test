// Package aggregator fans out search requests to all providers concurrently,
// retries transient failures with exponential backoff, enforces per-provider
// rate limits, and deduplicates results before returning them to the service layer.
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

// ProviderResult holds the outcome of one provider call, including any error
// and the wall-clock latency observed by the aggregator.
type ProviderResult struct {
	Name    string
	Flights []domain.Flight
	Err     error
	Latency time.Duration
}

// Config controls retry, timeout, and rate-limit behaviour for each provider.
type Config struct {
	// PerProviderTimeout is the maximum time to wait for a single provider,
	// including all retry attempts.
	PerProviderTimeout time.Duration
	// MaxRetries is the number of additional attempts after the first failure.
	MaxRetries int
	// BackoffBase is the base delay for exponential backoff between retries.
	BackoffBase time.Duration
	// BackoffMax caps the computed backoff delay.
	BackoffMax time.Duration
	// RateLimitCapacity is the token-bucket burst size per provider.
	RateLimitCapacity int
	// RateLimitRPS is the sustained request rate in requests per second per provider.
	RateLimitRPS float64
}

// DefaultConfig returns a [Config] suitable for development and testing:
// 2 s per-provider timeout, 2 retries, 100 ms base backoff, 20 rps rate limit
// with a burst of 10.
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

// Aggregator fans out search requests to a set of providers concurrently.
// It is safe for concurrent use by multiple goroutines.
type Aggregator struct {
	cfg       Config
	providers []providers.Provider
	limiters  map[string]*Limiter
}

// New creates an [Aggregator] that queries ps using the given cfg.
// One token-bucket [Limiter] is created per provider.
func New(cfg Config, ps []providers.Provider) *Aggregator {
	limiters := make(map[string]*Limiter, len(ps))
	for _, p := range ps {
		limiters[p.Name()] = NewLimiter(cfg.RateLimitCapacity, cfg.RateLimitRPS)
	}
	return &Aggregator{cfg: cfg, providers: ps, limiters: limiters}
}

// Aggregate queries all providers concurrently and returns one [ProviderResult]
// per provider. Slow or failing providers do not block results from others.
// Each provider call respects the per-provider timeout in [Config] and is
// retried up to [Config.MaxRetries] times on transient errors.
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

// callWithRetry calls p.Fetch with a per-provider timeout and retries on
// transient errors using exponential backoff with full jitter.
// It returns immediately on context cancellation or deadline exceeded.
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

// backoff returns a random delay in [0, min(base<<(attempt-1), max)] — full jitter.
func backoff(base, max time.Duration, attempt int) time.Duration {
	d := base << (attempt - 1)
	if d <= 0 || d > max {
		d = max
	}
	return time.Duration(rand.Int63n(int64(d) + 1))
}
