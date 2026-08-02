# go-listings-service

A small, production-shaped REST microservice in Go for managing property
listings. Backed by PostgreSQL with clean, testable layering, and publishing
domain events to Kafka on every listing change.

Built as a focused demonstration of Go backend engineering: clean architecture,
unit and integration tests, Docker, CI, and light self-built observability.

## Status

Work in progress. Milestones:

- [x] M0 Skeleton: module, layout, chi router, `/healthz`, graceful shutdown.
- [x] M1 DB + model: Postgres, migrations, pgxpool, repository.
- [x] M2 REST CRUD: service layer, handlers, validation, pagination/filters.
- [ ] M3 Tests: unit (fakes) + integration (testcontainers).
- [ ] M4 Kafka events: publish domain events on mutations.
- [ ] M5 Container + CI: multi-stage Dockerfile, GitHub Actions.
- [ ] M6 Polish: docs, metrics, structured logging.

## Quick start

```bash
cp .env.example .env
make up           # start Postgres (docker compose)
make run          # applies migrations, then serves the API on :8080
curl localhost:8080/healthz
open http://localhost:8080/docs   # interactive API documentation
```

## API documentation

OpenAPI 3.1 is generated from the code (via [Huma](https://huma.rocks/)), so it
never drifts from the implementation:

- Interactive docs (Stoplight Elements): `GET /docs`
- OpenAPI spec: `GET /openapi.json` and `GET /openapi.yaml`

If host port 5432 is already in use, start Postgres on another port and point
the app at it:

```bash
POSTGRES_HOST_PORT=5433 make up
DATABASE_URL="postgres://listings:listings@localhost:5433/listings?sslmode=disable" make run
```

Migrations are embedded in the binary and applied automatically on startup.
This keeps local dev and tests simple. In a multi-replica production deploy you
would instead run migrations as a separate, one-off step to avoid replicas
racing to migrate.

## Architecture

```
cmd/api/main.go            wire config, server, graceful shutdown
internal/config            env configuration
internal/listing           domain: model, validation, service (business logic)
internal/listing/store     repository over pgx (interface + postgres impl)
internal/events            kafka publisher (interface + kafka-go impl; noop for tests)
internal/httpapi           Huma operations, router, middleware (logging, recovery, request id)
internal/observability     healthz, readyz, metrics
migrations/                *.up.sql / *.down.sql
```

The `service` layer depends on `store` and `events` through interfaces, which
keeps business logic unit-testable with fakes.

## Endpoints

### Operational

| Endpoint   | Purpose                          |
|------------|----------------------------------|
| `/healthz` | Liveness probe                   |
| `/readyz`  | Readiness probe (deps reachable) |

Prometheus `/metrics` arrives in M6.

### Listings API (`/v1/listings`)

| Method & path                    | Purpose                                             |
|----------------------------------|-----------------------------------------------------|
| `POST /v1/listings`              | Create a listing (starts as `draft`) -> 201         |
| `GET /v1/listings/{id}`          | Fetch one -> 200 / 404                               |
| `GET /v1/listings`               | List with pagination + filters -> 200               |
| `PATCH /v1/listings/{id}`        | Partial update -> 200                                |
| `POST /v1/listings/{id}/publish` | Transition `draft` -> `published` -> 200 / 409       |
| `POST /v1/listings/{id}/sell`    | Transition `published` -> `sold` -> 200 / 409        |
| `DELETE /v1/listings/{id}`       | Delete -> 204 / 404                                  |

List query params: `limit` (1-100, default 20), `offset` (>=0), `city`,
`status` (`draft`/`published`/`sold`), `min_price`, `max_price` (cents).

Errors use a consistent envelope:

```json
{ "error": { "code": "validation_failed", "message": "one or more fields are invalid",
             "fields": { "title": "is required" } } }
```

Example:

```bash
curl -X POST localhost:8080/v1/listings -H 'Content-Type: application/json' -d '{
  "title": "Sunny 2-room", "price_cents": 250000, "city": "Toulouse",
  "postal_code": "31000", "surface_m2": 45, "rooms": 2,
  "seller_id": "35d22b88-77de-4c04-bbff-523e309ff93a"
}'
```
