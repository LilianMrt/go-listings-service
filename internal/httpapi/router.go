// Package httpapi wires the HTTP router, middleware, and handlers.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/LilianMrt/go-listings-service/internal/listing"
	"github.com/LilianMrt/go-listings-service/internal/observability"
)

// Deps holds the collaborators the router needs.
type Deps struct {
	Logger   *slog.Logger
	Health   *observability.Health
	Listings *listing.Service
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

	// Versioned API.
	lh := &listingHandler{svc: deps.Listings, logger: deps.Logger}
	r.Mount("/v1/listings", lh.routes())

	return r
}
