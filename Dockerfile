# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
FROM golang:1.25-alpine AS build
WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static, stripped binary: CGO off so it runs on a scratch/distroless base.
# Migrations are embedded, so the binary is fully self-contained.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Final stage -------------------------------------------------------------
# distroless/static: no shell, no package manager, runs as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
