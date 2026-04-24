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

type Cache struct {
	mu    sync.RWMutex
	items map[string]entry
	ttl   time.Duration
}

func New(ttl time.Duration) *Cache {
	c := &Cache{items: make(map[string]entry), ttl: ttl}
	go c.sweep()
	return c
}

func (c *Cache) Get(key string) (domain.SearchResponse, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		return domain.SearchResponse{}, false
	}
	return e.resp, true
}

func (c *Cache) Set(key string, resp domain.SearchResponse) {
	c.mu.Lock()
	c.items[key] = entry{resp: resp, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

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
