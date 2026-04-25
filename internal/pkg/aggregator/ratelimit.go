package aggregator

import (
	"context"
	"sync"
	"time"
)

// Limiter is a token-bucket rate limiter. It is safe for concurrent use
// by multiple goroutines.
type Limiter struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	last     time.Time
}

// NewLimiter creates a [Limiter] with the given burst capacity and sustained
// rate in tokens per second. The bucket starts full.
func NewLimiter(capacity int, rate float64) *Limiter {
	return &Limiter{
		tokens:   float64(capacity),
		capacity: float64(capacity),
		rate:     rate,
		last:     time.Now(),
	}
}

// Wait blocks until a token is available or ctx is cancelled.
// It returns ctx.Err() if the context is done before a token can be acquired.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(l.last).Seconds()
		l.tokens = minF(l.capacity, l.tokens+elapsed*l.rate)
		l.last = now
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		deficit := 1 - l.tokens
		wait := time.Duration(deficit / l.rate * float64(time.Second))
		l.mu.Unlock()

		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
