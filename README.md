# BookCabin Flight Search & Aggregation

Go implementation of the BookCabin take-home test. Aggregates flights from four
mock airline providers, normalizes heterogeneous response shapes into a single
schema, and exposes the result over HTTP.

## Requirements

- Go 1.22+

## Run

```bash
cd bookcabin
go mod tidy
go run ./cmd/server           # listens on :8080
go run ./cmd/server -addr :9000 -cache-ttl 1m
```

Health check:

```bash
curl http://localhost:8080/healthz
```

## Search

```bash
curl -s http://localhost:8080/search -X POST -H 'content-type: application/json' -d '{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "passengers": 1,
  "cabinClass": "economy"
}' | jq
```

### With filters and sort

```bash
curl -s http://localhost:8080/search -X POST -H 'content-type: application/json' -d '{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "passengers": 1,
  "cabinClass": "economy",
  "filters": {
    "max_price": 1000000,
    "max_stops": 0,
    "airlines": ["GA", "JT"],
    "departure_window": { "from_hour": 6, "to_hour": 12 }
  },
  "sort_by": "best_value",
  "sort_order": "asc"
}' | jq
```

### Round-trip

```bash
curl -s http://localhost:8080/search -X POST -H 'content-type: application/json' -d '{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "returnDate": "2025-12-20",
  "passengers": 1,
  "cabinClass": "economy"
}' | jq
```

## Tests

```bash
go test ./...
go test -race ./...
```

## Layout

```
cmd/server/          HTTP entry point
internal/domain/     unified Flight/SearchRequest models + validation
internal/airport/    airport → city/TZ lookup
internal/providers/  4 provider adapters, each with its own normalizer
internal/aggregator/ parallel fanout, exp-backoff retry, token-bucket rate limit, dedup
internal/filter/     price/stops/airline/duration/time-window filters + sort
internal/ranker/     "best value" scoring
internal/cache/      TTL in-memory cache keyed on search params
internal/service/    orchestrator
internal/api/        HTTP handler
internal/money/      IDR formatting
```

## Design notes

**Separation of concerns.** Each layer has one job: providers fetch+normalize,
aggregator handles fan-out and transient failures, filter/sort/rank operate on
the unified model, service orchestrates, API translates HTTP. A new provider
needs one file in `internal/providers/`; nothing else changes.

**Data inconsistency handling.** Each provider ships its own time format:
- AirAsia: RFC3339 with colon offset (`+07:00`)
- Batik Air: compact offset (`+0700`)
- Garuda: RFC3339
- Lion Air: naive datetime + named IANA timezone

Normalizers convert all to RFC3339 with a Unix timestamp. When a flight's
top-level fields contradict nested segment data (e.g. Garuda's `GA315` shows
`arrival.airport: SUB` but its `segments[]` terminate at `DPS`), the
normalizer rebuilds departure/arrival/stops from the segment chain. Wall-clock
duration (derived from timestamps) is preferred over the provider-supplied
`duration_minutes` / `flight_time` / `travel_time` fields to absorb
TZ-accounting mistakes in the source data.

Every normalized flight passes `Flight.Validate()`: arrival > departure,
non-negative stops, positive price and duration. Invalid rows are dropped
silently (a production build would emit a metric).

**Parallelism + resilience.** `Aggregator.Aggregate` fans out one goroutine
per provider. Per-provider context timeout (2s default) caps how long any one
upstream can block. Exponential backoff with full jitter on retry (2 attempts
default), cancelled eagerly when the parent context is done. AirAsia's
10%-failure mock exercises this path.

**Rate limiting.** A token-bucket `Limiter` per provider caps outbound call
rate (20 rps, burst 10 by default). Sharing one limiter per provider means
bursts from concurrent search requests still get smoothed.

**Caching.** Search responses are cached by `(origin, destination, date,
passengers, cabin, returnDate)`. Filters and sort are intentionally excluded
so two requests with different UI filters share the same upstream fetch.
Default TTL 30s; short because prices change.

**Dedup.** After aggregation, flights with the same airline code + flight
number + departure date are collapsed to the cheapest — satisfies the "compare
prices across providers for the same flight" requirement.

**Best-value ranking.** `ranker.Rank` builds a composite score from price
(60%), duration (30%), and stops (10%) normalized against the current result
set, giving a 0..1 score per flight. Exposed as `price.best_value_score` and
usable via `sort_by: best_value`.

**IDR formatting.** Prices are emitted both as a raw integer amount and a
localized string (`"Rp 1.250.000"`).

**Bonus items delivered:**
- Best-value scoring
- Round-trip (outbound + inbound response sections)
- WIB/WITA/WIT handled via IANA names (`Asia/Jakarta` / `Asia/Makassar` /
  `Asia/Jayapura`)
- Per-provider token-bucket rate limiting
- Exponential backoff with full jitter + retry
- IDR thousands-separator formatting
- Parallel provider queries with timeout
- Graceful HTTP shutdown

Not implemented: multi-city search (the single-leg model covers it naturally
as a sequence of one-way searches but the API surface isn't exposed).

## Sample request/response shapes

Request body (one-way):

```json
{ "origin": "CGK", "destination": "DPS", "departureDate": "2025-12-15", "passengers": 1, "cabinClass": "economy" }
```

Response is wrapped in `{ "outbound": ..., "inbound": ... }` (inbound omitted
for one-way). Each leg matches the unified schema in the task spec.
