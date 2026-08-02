// Package httpapi wires the HTTP router, middleware, and handlers.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/LilianMrt/go-listings-service/internal/listing"
	"github.com/LilianMrt/go-listings-service/internal/observability"
)

// apiVersion is the version reported in the generated OpenAPI document.
const apiVersion = "0.1.0"

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

	// Operational endpoints, unversioned and outside the documented API.
	r.Get("/healthz", deps.Health.Livez)
	r.Get("/readyz", deps.Health.Readyz)

	// Huma wraps the chi router: it generates OpenAPI 3.1 from the registered
	// operations and serves interactive docs at /docs (Stoplight Elements),
	// with the spec at /openapi.json and /openapi.yaml.
	config := huma.DefaultConfig("go-listings-service", apiVersion)
	config.Info.Description = "REST API for managing property listings, backed by PostgreSQL."
	api := humachi.New(r, config)

	registerListings(api, deps.Listings, deps.Logger)

	return r
}
