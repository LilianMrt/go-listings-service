// Package httpapi wires the HTTP router, middleware, and handlers.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/LilianMrt/go-listings-service/internal/observability"
)

// Deps holds the collaborators the router needs. It grows as milestones land
// (listing service, etc.); for M0 it carries observability only.
type Deps struct {
	Logger *slog.Logger
	Health *observability.Health
}

// NewRouter builds the application router with middleware and routes.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Recoverer)

	// Operational endpoints (unversioned, outside /v1).
	// /metrics (Prometheus) is deliberately deferred to M6 to avoid pulling a
	// heavy dependency before the service does anything worth measuring.
	r.Get("/healthz", deps.Health.Livez)
	r.Get("/readyz", deps.Health.Readyz)

	return r
}
