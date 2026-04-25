// Package airlinemock provides HTTP handlers that simulate each airline's API server.
// Each handler injects a realistic latency before responding and serves the JSON
// data supplied via its Data field. Handlers are used by [cmd/server] to start
// dedicated mock servers on random local ports.
package airlinemock

import (
	"math/rand"
	"net/http"
	"time"
)

// GarudaHandler simulates Garuda Indonesia's API: fast response (50–100 ms).
type GarudaHandler struct {
	// Data is the raw JSON response body to serve.
	Data []byte
}

// ServeHTTP waits a randomized 50–100 ms delay then writes Data as JSON.
// If the request context is cancelled during the delay, the handler returns without writing.
func (h GarudaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 50*time.Millisecond, 100*time.Millisecond)
	writeJSON(w, h.Data)
}

// LionHandler simulates Lion Air's API: medium response (100–200 ms).
type LionHandler struct {
	// Data is the raw JSON response body to serve.
	Data []byte
}

// ServeHTTP waits a randomized 100–200 ms delay then writes Data as JSON.
func (h LionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 100*time.Millisecond, 200*time.Millisecond)
	writeJSON(w, h.Data)
}

// BatikHandler simulates Batik Air's API: slow response (200–400 ms).
type BatikHandler struct {
	// Data is the raw JSON response body to serve.
	Data []byte
}

// ServeHTTP waits a randomized 200–400 ms delay then writes Data as JSON.
func (h BatikHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 200*time.Millisecond, 400*time.Millisecond)
	writeJSON(w, h.Data)
}

// AirAsiaHandler simulates AirAsia's API: fast but occasionally unavailable (50–150 ms).
//
// FailureRate controls how often the handler returns HTTP 503:
//   - 0: default 10% failure rate
//   - < 0: never fails
//   - > 0: fails at exactly that fraction (e.g. 1.0 = always fail)
type AirAsiaHandler struct {
	Data        []byte
	FailureRate float64
}

// ServeHTTP waits a randomized 50–150 ms delay, then either returns HTTP 503
// according to FailureRate or writes Data as JSON.
func (h AirAsiaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 50*time.Millisecond, 150*time.Millisecond)
	rate := h.FailureRate
	if rate < 0 {
		rate = 0
	} else if rate == 0 {
		rate = 0.10
	}
	if rand.Float64() < rate {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, h.Data)
}

// sleep blocks for a random duration in [min, max], or returns early if r's
// context is cancelled.
func sleep(r *http.Request, min, max time.Duration) bool {
	d := min + time.Duration(rand.Int63n(int64(max-min+1)))
	select {
	case <-time.After(d):
		return true
	case <-r.Context().Done():
		return false
	}
}

func writeJSON(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
