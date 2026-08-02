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
- [x] M3 Tests: unit (fakes) + integration (testcontainers).
- [x] M4 Kafka events: publish domain events on mutations.
- [ ] M5 Container + CI: multi-stage Dockerfile, GitHub Actions.
- [ ] M6 Polish: docs, metrics, structured logging.

## Quick start

```bash
cp .env.example .env
make up           # start Postgres + Kafka (docker compose)
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

## Testing

```bash
make test-unit   # fast unit tests, no Docker (service logic with fakes)
make test        # everything, incl. integration tests via testcontainers (needs Docker)
```

Unit tests cover the service layer with a fake repository (validation, status
transitions, partial updates). Integration tests spin a throwaway Postgres with
[testcontainers-go](https://golang.testcontainers.org/): the repository CRUD and
the full HTTP lifecycle (Huma -> service -> Postgres) run unattended, no manually
started database.

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

## Events

Every mutation publishes a domain event to the Kafka topic `listings.events`,
keyed by the listing id so all events for one listing stay ordered on the same
partition. Event types: `listing.created`, `listing.updated` (updates and
status transitions), `listing.deleted`.

```json
{
  "type": "listing.created",
  "id": "b6310b10-9636-4d5a-a977-6f85e558cf04",
  "occurred_at": "2026-08-02T14:04:38Z",
  "data": { "id": "...", "title": "...", "price_cents": 250000, "status": "draft", "...": "..." }
}
```

Design choices, and their tradeoffs:

- **Emitted after the DB commit, best-effort.** The event schema is a contract,
  so it lives in `internal/events` decoupled from the HTTP DTO. Publishing
  happens only once the write is durable.
- **At-least-once, no transactional outbox (v1).** If the process dies between
  the commit and the publish, an event can be missed; if a retry succeeds after
  a timeout, it can be duplicated. Consumers should be idempotent. A production
  system needing exactly-once-ish delivery would add an outbox table drained by
  a relay. Kept out of scope here deliberately.
- **A publish failure never fails the request** (the data is already saved); it
  is logged instead.
- The topic is auto-created on first write for local/dev convenience (with a
  short produce retry so the first event is not lost to the creation race); in
  production it would be provisioned ahead of time.

Set `KAFKA_ENABLED=false` to run without a broker (events are discarded).
