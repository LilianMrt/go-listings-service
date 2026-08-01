// Package observability provides light, self-built operational endpoints:
// liveness (/healthz) and readiness (/readyz).
package observability

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Health tracks service readiness. Liveness is always true once the process is
// up; readiness flips to true only once the database is reachable.
type Health struct {
	ready atomic.Bool
}

func NewHealth() *Health {
	return &Health{}
}

// SetReady marks the service ready (or not) to serve traffic.
func (h *Health) SetReady(ready bool) {
	h.ready.Store(ready)
}

// Livez reports process liveness. If the handler responds, the process is alive.
func (h *Health) Livez(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz reports whether dependencies are ready.
func (h *Health) Readyz(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
