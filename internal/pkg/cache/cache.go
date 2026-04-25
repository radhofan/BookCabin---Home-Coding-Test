// Package cache provides an in-memory TTL cache for search responses.
// Responses are keyed on the core search parameters (origin, destination, date,
// passengers, cabin class, return date). Filters and sort order are intentionally
// excluded from the key so that two requests differing only in presentation
// share a single upstream fetch.
package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"bookcabin/internal/pkg/domain"
)

type entry struct {
	resp domain.SearchResponse
	exp  time.Time
}

// Cache is a thread-safe in-memory store for [domain.SearchResponse] values
// with per-entry expiry. Expired entries are evicted in the background.
type Cache struct {
	mu    sync.RWMutex
	items map[string]entry
	ttl   time.Duration
}

// New creates a [Cache] where every stored entry expires after ttl.
// A background goroutine sweeps expired entries at each ttl interval.
func New(ttl time.Duration) *Cache {
	c := &Cache{items: make(map[string]entry), ttl: ttl}
	go c.sweep()
	return c
}

// Get returns the cached [domain.SearchResponse] for key and reports whether
// a valid (non-expired) entry was found.
func (c *Cache) Get(key string) (domain.SearchResponse, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		return domain.SearchResponse{}, false
	}
	return e.resp, true
}

// Set stores resp under key, overwriting any previous value.
// The entry expires after the TTL configured in [New].
func (c *Cache) Set(key string, resp domain.SearchResponse) {
	c.mu.Lock()
	c.items[key] = entry{resp: resp, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

// sweep runs in the background and deletes expired entries once per TTL interval.
func (c *Cache) sweep() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.items {
			if now.After(v.exp) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

// Key derives the cache key for req. Only the core search fields are included;
// filters and sort options are excluded so filtered requests can share a cache entry.
func Key(req domain.SearchRequest) string {
	parts := []string{
		strings.ToUpper(req.Origin),
		strings.ToUpper(req.Destination),
		req.DepartureDate,
		fmt.Sprintf("p=%d", req.Passengers),
		strings.ToLower(req.CabinClass),
	}
	if req.ReturnDate != nil {
		parts = append(parts, *req.ReturnDate)
	}
	return strings.Join(parts, "|")
}
