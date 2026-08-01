# go-listings-service

A small, production-shaped REST microservice in Go for managing property
listings. Backed by PostgreSQL with clean, testable layering, and publishing
domain events to Kafka on every listing change.

Built as a focused demonstration of Go backend engineering: clean architecture,
unit and integration tests, Docker, CI, and light self-built observability.

## Status

Work in progress. Milestones:

- [x] M0 Skeleton: module, layout, chi router, `/healthz`, graceful shutdown.
- [ ] M1 DB + model: Postgres, migrations, pgxpool, repository.
- [ ] M2 REST CRUD: service layer, handlers, validation, pagination/filters.
- [ ] M3 Tests: unit (fakes) + integration (testcontainers).
- [ ] M4 Kafka events: publish domain events on mutations.
- [ ] M5 Container + CI: multi-stage Dockerfile, GitHub Actions.
- [ ] M6 Polish: docs, metrics, structured logging.

## Quick start

```bash
cp .env.example .env
make run          # starts the API on :8080
curl localhost:8080/healthz
```

## Architecture

```
cmd/api/main.go            wire config, server, graceful shutdown
internal/config            env configuration
internal/listing           domain: model, validation, service (business logic)
internal/listing/store     repository over pgx (interface + postgres impl)
internal/events            kafka publisher (interface + kafka-go impl; noop for tests)
internal/httpapi           handlers, router, middleware (logging, recovery, request id)
internal/observability     healthz, readyz, metrics
migrations/                *.up.sql / *.down.sql
```

The `service` layer depends on `store` and `events` through interfaces, which
keeps business logic unit-testable with fakes.

## Endpoints

| Endpoint   | Purpose                          |
|------------|----------------------------------|
| `/healthz` | Liveness probe                   |
| `/readyz`  | Readiness probe (deps reachable) |

The versioned `/v1/listings` API arrives in M2; Prometheus `/metrics` in M6.
