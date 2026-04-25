package airlinemock

import (
	"math/rand"
	"net/http"
	"time"
)

// GarudaHandler simulates Garuda Indonesia: fast response (50-100ms).
type GarudaHandler struct{ Data []byte }

func (h GarudaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 50*time.Millisecond, 100*time.Millisecond)
	writeJSON(w, h.Data)
}

// LionHandler simulates Lion Air: medium response (100-200ms).
type LionHandler struct{ Data []byte }

func (h LionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 100*time.Millisecond, 200*time.Millisecond)
	writeJSON(w, h.Data)
}

// BatikHandler simulates Batik Air: slow response (200-400ms).
type BatikHandler struct{ Data []byte }

func (h BatikHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sleep(r, 200*time.Millisecond, 400*time.Millisecond)
	writeJSON(w, h.Data)
}

// AirAsiaHandler simulates AirAsia: fast but occasionally fails.
// FailureRate: 0 = default 10%, <0 = never fail, >0 = use that rate.
type AirAsiaHandler struct {
	Data        []byte
	FailureRate float64
}

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
