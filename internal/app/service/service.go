// Package service orchestrates a flight search: it checks the cache, fans out
// to providers via the aggregator, deduplicates results, applies presentation
// logic (filtering, ranking, sorting, IDR formatting), and writes back to cache.
package service

import (
	"context"
	"time"

	"bookcabin/internal/pkg/aggregator"
	"bookcabin/internal/pkg/cache"
	"bookcabin/internal/pkg/domain"
	"bookcabin/internal/pkg/filter"
	"bookcabin/internal/pkg/money"
	"bookcabin/internal/pkg/ranker"
)

// Service orchestrates flight searches. It is safe for concurrent use by
// multiple goroutines.
type Service struct {
	agg   *aggregator.Aggregator
	cache *cache.Cache
}

// New creates a [Service] backed by the given aggregator and cache.
func New(agg *aggregator.Aggregator, c *cache.Cache) *Service {
	return &Service{agg: agg, cache: c}
}

// TripResponse is the top-level response returned by POST /search.
// Inbound is non-nil only for round-trip requests.
type TripResponse struct {
	Outbound domain.SearchResponse  `json:"outbound"`
	Inbound  *domain.SearchResponse `json:"inbound,omitempty"`
}

// Search validates req, searches for the outbound leg, and — when ReturnDate is
// set — searches for the inbound leg with origin and destination swapped.
// It returns a [TripResponse] containing one or both legs.
func (s *Service) Search(ctx context.Context, req domain.SearchRequest) (TripResponse, error) {
	if err := req.Validate(); err != nil {
		return TripResponse{}, err
	}

	outbound, err := s.searchLeg(ctx, req)
	if err != nil {
		return TripResponse{}, err
	}

	if req.ReturnDate == nil || *req.ReturnDate == "" {
		return TripResponse{Outbound: outbound}, nil
	}

	ret := req
	ret.Origin, ret.Destination = req.Destination, req.Origin
	d := *req.ReturnDate
	ret.DepartureDate = d
	ret.ReturnDate = nil

	inbound, err := s.searchLeg(ctx, ret)
	if err != nil {
		return TripResponse{}, err
	}
	return TripResponse{Outbound: outbound, Inbound: &inbound}, nil
}

// searchLeg performs a single-leg search: cache lookup → aggregation → dedup →
// cache write → filter/rank/sort/format.
// Raw (pre-presentation) flights are cached so that re-requests with different
// filters still benefit from the cached upstream data.
func (s *Service) searchLeg(ctx context.Context, req domain.SearchRequest) (domain.SearchResponse, error) {
	start := time.Now()

	base := domain.SearchResponse{
		SearchCriteria: domain.SearchCriteria{
			Origin:        req.Origin,
			Destination:   req.Destination,
			DepartureDate: req.DepartureDate,
			Passengers:    req.Passengers,
			CabinClass:    req.CabinClass,
		},
	}

	key := cache.Key(req)
	if cached, ok := s.cache.Get(key); ok {
		resp := cached
		resp.Metadata.CacheHit = true
		resp.Flights = applyPresentation(cached.Flights, req)
		resp.Metadata.TotalResults = len(resp.Flights)
		resp.Metadata.SearchTimeMs = time.Since(start).Milliseconds()
		return resp, nil
	}

	results := s.agg.Aggregate(ctx, req)
	var flights []domain.Flight
	succeeded, failed := 0, 0
	for _, r := range results {
		if r.Err != nil {
			failed++
			continue
		}
		succeeded++
		flights = append(flights, r.Flights...)
	}

	flights = aggregator.Dedup(flights)

	cacheResp := base
	cacheResp.Flights = flights
	cacheResp.Metadata = domain.Metadata{
		ProvidersQueried:   len(results),
		ProvidersSucceeded: succeeded,
		ProvidersFailed:    failed,
	}
	s.cache.Set(key, cacheResp)

	resp := base
	resp.Flights = applyPresentation(flights, req)
	resp.Metadata = domain.Metadata{
		TotalResults:       len(resp.Flights),
		ProvidersQueried:   len(results),
		ProvidersSucceeded: succeeded,
		ProvidersFailed:    failed,
		SearchTimeMs:       time.Since(start).Milliseconds(),
	}
	return resp, nil
}

// applyPresentation filters, ranks, sorts, and formats a copy of in according
// to the presentation options in req. It does not modify in.
func applyPresentation(in []domain.Flight, req domain.SearchRequest) []domain.Flight {
	flights := make([]domain.Flight, len(in))
	copy(flights, in)
	flights = filter.Apply(flights, req.Filters)
	ranker.Rank(flights, ranker.DefaultWeights())
	filter.Sort(flights, req.SortBy, req.SortOrder)
	for i := range flights {
		flights[i].Price.Formatted = money.Format(flights[i].Price.Amount, flights[i].Price.Currency)
	}
	return flights
}
