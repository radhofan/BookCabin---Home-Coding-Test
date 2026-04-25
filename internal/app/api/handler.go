// Package api implements the HTTP layer of the BookCabin server.
// It exposes exactly two endpoints: GET /healthz and POST /search.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"bookcabin/internal/app/service"
	"bookcabin/internal/pkg/domain"
)

// Handler holds the HTTP handlers for the BookCabin API.
type Handler struct {
	svc *service.Service
}

// New creates a [Handler] backed by svc.
func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Routes returns an [http.Handler] with all routes registered and wrapped
// in a request-logging middleware.
//
// Registered routes:
//   - POST /search — flight search and aggregation
//   - GET  /healthz — liveness probe
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /search", h.handleSearch)
	mux.HandleFunc("GET /healthz", h.handleHealth)
	return requestLogger(mux)
}

// handleHealth writes a 200 OK JSON response indicating the server is alive.
func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleSearch decodes the request body, delegates to [service.Service.Search],
// and writes the result as JSON. It returns 400 on a bad request body or
// validation error, and 504 if the search times out.
func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req domain.SearchRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.svc.Search(ctx, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		writeError(w, status, err)
		return
	}
	if resp.Inbound == nil {
		writeJSON(w, http.StatusOK, resp.Outbound)
	} else {
		writeJSON(w, http.StatusOK, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// requestLogger is middleware that logs the method, path, and elapsed time
// of every request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
