# BookCabin Flight Search & Aggregation

Go implementation of the BookCabin take-home test. Aggregates flights from four
mock airline providers, normalizes heterogeneous response shapes into a single
schema, and exposes the result over HTTP.

## Requirements

- Go 1.22+
- [jq](https://jqlang.github.io/jq/) (for pretty-printing API responses, optional)

```bash
scoop install jq        # Windows (Scoop)
brew install jq         # macOS
sudo apt install jq     # Ubuntu/Debian
```

## Run

```bash
cd BookCabin
go mod tidy
go run ./cmd/server
```

On startup the server automatically launches four airline mock servers on
random local ports and logs their addresses:

```
airline mock: Garuda Indonesia      http://127.0.0.1:54321
airline mock: Lion Air              http://127.0.0.1:54322
airline mock: Batik Air             http://127.0.0.1:54323
airline mock: AirAsia               http://127.0.0.1:54324
bookcabin listening on :8080
```

The main API is always on `:8080`. Airline servers pick free ports automatically
so there are no port conflicts.

Custom address or cache TTL:

```bash
go run ./cmd/server -addr :9000 -cache-ttl 1m
```

**Important:** run from inside the `BookCabin/` directory. The server reads
mock data from `test/testdata/` relative to the working directory.

## API

The backend exposes exactly two endpoints — nothing else.

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Health check — returns `{"status":"ok"}` |
| `POST` | `/search` | Flight search and aggregation |

```bash
curl http://localhost:8080/healthz
```

## Search

### One-way search

Returns all available flights from `origin` to `destination` on `departureDate`, normalized across all four providers.

| Param | Description |
|---|---|
| `origin` | Departure airport IATA code |
| `destination` | Arrival airport IATA code |
| `departureDate` | Date in `YYYY-MM-DD` format |
| `passengers` | Number of passengers (used to check seat availability) |
| `cabinClass` | `economy`, `business`, `first`, or `premium_economy` |

```bash
curl -s -X POST http://localhost:8080/search -H "content-type: application/json" -d "{\"origin\":\"CGK\",\"destination\":\"DPS\",\"departureDate\":\"2025-12-15\",\"passengers\":1,\"cabinClass\":\"economy\"}" | jq
```

### With filters and sort

Narrows results after aggregation and controls sort order — all filter/sort params are optional.

| Param | Description |
|---|---|
| `filters.max_price` | Drop flights above this IDR amount |
| `filters.max_stops` | `0` = direct only, `1` = max one stop, etc. |
| `filters.airlines` | Whitelist of airline IATA codes |
| `filters.departure_window` | Only flights departing between `from_hour` and `to_hour` (24h) |
| `sort_by` | `price`, `duration`, `departure_time`, `arrival_time`, or `best_value` |
| `sort_order` | `asc` or `desc` |

```bash
curl -s -X POST http://localhost:8080/search -H "content-type: application/json" -d "{\"origin\":\"CGK\",\"destination\":\"DPS\",\"departureDate\":\"2025-12-15\",\"passengers\":1,\"cabinClass\":\"economy\",\"filters\":{\"max_price\":1000000,\"max_stops\":0,\"airlines\":[\"GA\",\"JT\"],\"departure_window\":{\"from_hour\":6,\"to_hour\":12}},\"sort_by\":\"best_value\",\"sort_order\":\"asc\"}" | jq
```

### Round-trip

Same as one-way but also fetches the return leg — response contains both `outbound` and `inbound` flight arrays.

| Param | Description |
|---|---|
| `returnDate` | Return date in `YYYY-MM-DD` format |

```bash
curl -s -X POST http://localhost:8080/search -H "content-type: application/json" -d "{\"origin\":\"CGK\",\"destination\":\"DPS\",\"departureDate\":\"2025-12-15\",\"returnDate\":\"2025-12-20\",\"passengers\":1,\"cabinClass\":\"economy\"}" | jq
```

### POST /search — full parameter reference

#### Top-level fields

| Field | Type | Required | Description |
|---|---|---|---|
| `origin` | string | yes | Departure airport IATA code (e.g. `"CGK"`) |
| `destination` | string | yes | Arrival airport IATA code (e.g. `"DPS"`) |
| `departureDate` | string | yes | Date in `YYYY-MM-DD` format |
| `returnDate` | string | no | Return date in `YYYY-MM-DD`. Omit or set `null` for one-way |
| `passengers` | int | yes | Number of passengers (must be ≥ 1) |
| `cabinClass` | string | yes | See cabin class options below. Defaults to `economy` if omitted |
| `filters` | object | no | See filters below. Omit the whole object to skip filtering |
| `sort_by` | string | no | See sort options below. Defaults to `price` |
| `sort_order` | string | no | `asc` or `desc`. Defaults to `asc` |

#### `cabinClass` options

| Value | Description |
|---|---|
| `economy` | Economy class |
| `premium_economy` | Premium economy |
| `business` | Business class |
| `first` | First class |

#### `filters` fields

All filter fields are optional. Omit any field to skip that filter.

| Field | Type | Description |
|---|---|---|
| `min_price` | int | Drop flights cheaper than this IDR amount |
| `max_price` | int | Drop flights more expensive than this IDR amount |
| `max_stops` | int | `0` = direct only, `1` = max one stop, `2` = max two stops, etc. |
| `airlines` | string[] | Whitelist by IATA code (e.g. `["GA", "JT"]`) or full name. Only matching airlines are returned |
| `max_duration_minutes` | int | Drop flights with total duration longer than this |
| `departure_window` | object | Only flights departing within the hour range — see below |
| `arrival_window` | object | Only flights arriving within the hour range — see below |

#### Time window fields (`departure_window` / `arrival_window`)

| Field | Type | Description |
|---|---|---|
| `from_hour` | int | Start of window, 24h format (0–23) |
| `to_hour` | int | End of window, 24h format (0–23). Wraps midnight if `to_hour` < `from_hour` |

#### `sort_by` options

| Value | Description |
|---|---|
| `price` | Sort by ticket price (default) |
| `duration` | Sort by total flight duration |
| `departure_time` | Sort by departure timestamp |
| `arrival_time` | Sort by arrival timestamp |
| `best_value` | Sort by composite score: price 60%, duration 30%, stops 10% |

## Tests

```bash
go test ./...
go test -race ./...
```

## Project Standards

Follows the [golang-standards/project-layout](https://github.com/golang-standards/project-layout) convention.
All doc comments follow the official [Go doc comment](https://go.dev/doc/comment) style.

## How the mock airline servers work

Each airline runs as a real HTTP server in the same process on a random port.
The providers call them over localhost HTTP — no shared memory, no function
calls. This mirrors a real multi-service architecture.

| Airline          | Simulated latency | Failure rate |
|------------------|-------------------|--------------|
| Garuda Indonesia | 50–100 ms         | none         |
| Lion Air         | 100–200 ms        | none         |
| Batik Air        | 200–400 ms        | none         |
| AirAsia          | 50–150 ms         | 10%          |

AirAsia failures return HTTP 503. The aggregator retries with exponential
backoff (2 attempts, full jitter), so transient failures are recovered
automatically.

## Design notes

**Separation of concerns.** The codebase is split into thin, single-responsibility layers: providers own HTTP transport and normalization, the aggregator owns fan-out and resilience, filter/ranker own result presentation, the service orchestrates them, and the API handler owns HTTP encoding. Adding a new provider requires one file in `internal/pkg/providers/` and one handler in `internal/pkg/airlinemock/` — nothing else changes.

**Data inconsistency handling.** Each provider ships a different time format: Garuda uses RFC3339, Batik Air uses a compact numeric offset (`+0700`), AirAsia uses RFC3339 with colon offset (`+07:00`), and Lion Air uses naive local datetime strings paired with a named IANA timezone. Each normalizer parses its own format and converts to RFC3339 + Unix timestamp. When a provider's top-level arrival field contradicts its segment chain (e.g. Garuda's `GA315` lists `arrival.airport: SUB` but the segments terminate at `DPS`), the normalizer discards the top-level field and rebuilds departure, arrival, and stop count directly from the segment chain. Wall-clock duration computed from timestamps is always preferred over the provider-supplied `duration_minutes`, `flight_time`, or `travel_time` fields, which may contain timezone-accounting errors. Every normalized flight is passed through `Flight.Validate()` — checking that arrival is after departure, price and duration are positive, and stops are non-negative — and silently dropped if it fails.

**Parallelism + resilience.** `Aggregator.Aggregate` launches one goroutine per provider using a `sync.WaitGroup`. Each goroutine runs under a separate per-provider context with a 2 s timeout, so a slow provider cannot block results from the others. Failed calls are retried up to 2 times using exponential backoff with full jitter, and the retry loop short-circuits immediately on context cancellation or deadline exceeded. AirAsia's 10% HTTP 503 failure rate exercises this retry path on every run.

**Rate limiting.** A token-bucket `Limiter` (20 rps, burst 10) is created per provider at startup and shared across all concurrent requests to that provider. Tokens are replenished continuously based on elapsed time, and `Wait` blocks until a token is available or the context is cancelled.

**Caching.** Raw aggregated results (before filtering and sorting) are cached under a key composed of `origin|destination|date|passengers|cabinClass|returnDate`. Filters and sort order are excluded from the key deliberately — two requests that differ only in presentation parameters share the same upstream fetch and apply their own presentation layer on top of the cached data. Default TTL is 30 s.

**Dedup.** After aggregation, flights sharing the same airline IATA code, flight number, and departure day are collapsed to the single cheapest option. The key uses `departure.timestamp / 86400` (seconds per day) to normalize across timezone-aware timestamps.

**Best-value ranking.** `ranker.Rank` min-max normalizes price, duration, and stops independently across the current result set, producing a 0–1 score for each dimension where 1 is best. The three scores are combined as a weighted sum (price 60%, duration 30%, stops 10%) and stored as `price.best_value_score`. Because normalization is relative, scores shift when filters reduce the result set — a filtered set recalculates against its own min/max.

**IDR formatting.** `money.FormatIDR` formats amounts with dot thousands separators and an `Rp` prefix (e.g. `1250000` → `"Rp 1.250.000"`). The raw integer amount is always present alongside the formatted string so callers can sort or compare numerically.

## Bonus items

**Round-trip search.** When `returnDate` is set, the service calls `searchLeg` twice — once for the outbound leg and once with origin and destination swapped for the inbound leg. Both legs go through the full aggregation, dedup, cache, and presentation pipeline independently. The response wraps both as `outbound` and `inbound`.

**Timezone handling.** Lion Air supplies naive datetime strings (no offset) paired with a named IANA timezone string such as `Asia/Jakarta`, `Asia/Makassar`, or `Asia/Jayapura`. The normalizer calls `time.ParseInLocation` with the resolved `*time.Location`, correctly handling WIB (UTC+7), WITA (UTC+8), and WIT (UTC+9). The airport lookup table provides a fallback timezone for any provider that omits it entirely.

**Per-provider rate limiting.** Each provider gets its own token-bucket limiter so that burst traffic from concurrent search requests is smoothed independently per airline, preventing any single provider from being flooded.

**Exponential backoff with retry.** Failed provider calls are retried with delays computed as `rand.Int63n(min(base << (attempt-1), max) + 1)` — full jitter over a doubling window, capped at 1 s. The retry loop is wired to the per-provider context so retries are abandoned as soon as the timeout fires.

**Parallel provider queries with timeout.** All four providers are queried concurrently via goroutines. Each runs under a 2 s context deadline independent of the others, so the slowest provider (Batik Air at up to 400 ms) determines response time only in the worst case, not the sum of all provider latencies.

**Graceful shutdown.** The main server listens for `SIGINT` / `SIGTERM` and calls `srv.Close()`, allowing in-flight requests to drain before the process exits.

**Not implemented:** multi-city search. The single-leg model naturally supports it as a sequence of one-way searches, but the API surface to express an arbitrary city chain is not exposed.

## Sample response

```bash
curl -s -X POST http://localhost:8080/search -H "content-type: application/json" -d "{\"origin\":\"CGK\",\"destination\":\"DPS\",\"departureDate\":\"2025-12-15\",\"passengers\":1,\"cabinClass\":\"economy\"}" | jq
```

One-way search CGK → DPS, 13 flights returned (one shown for simplicity).

```json
{
  "search_criteria": {
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "passengers": 1,
    "cabin_class": "economy"
  },
  "metadata": {
    "total_results": 13,
    "providers_queried": 4,
    "providers_succeeded": 4,
    "providers_failed": 0,
    "search_time_ms": 311,
    "cache_hit": false
  },
  "flights": [
    {
      "id": "QZ532_AirAsia",
      "provider": "AirAsia",
      "airline": {
        "name": "AirAsia",
        "code": "QZ"
      },
      "flight_number": "QZ532",
      "departure": {
        "airport": "CGK",
        "city": "Jakarta",
        "datetime": "2025-12-15T19:30:00+07:00",
        "timestamp": 1765801800
      },
      "arrival": {
        "airport": "DPS",
        "city": "Denpasar",
        "datetime": "2025-12-15T22:10:00+08:00",
        "timestamp": 1765807800
      },
      "duration": {
        "total_minutes": 100,
        "formatted": "1h 40m"
      },
      "stops": 0,
      "price": {
        "amount": 595000,
        "currency": "IDR",
        "formatted": "Rp 595.000",
        "best_value_score": 0.9516
      },
      "available_seats": 72,
      "cabin_class": "economy",
      "aircraft": null,
      "amenities": [],
      "baggage": {
        "carry_on": "Cabin baggage only",
        "checked": "Additional fee"
      }
    }
  ]
}
```