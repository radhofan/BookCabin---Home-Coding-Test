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

**Separation of concerns.** Each layer has one job: providers fetch+normalize,
aggregator handles fan-out and transient failures, filter/sort/rank operate on
the unified model, service orchestrates, API translates HTTP. A new provider
needs one file in `internal/pkg/providers/`; nothing else changes.

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

Note: multi-city search (the single-leg model covers it naturally
as a sequence of one-way searches but the API surface isn't exposed).