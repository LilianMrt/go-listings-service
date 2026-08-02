.DEFAULT_GOAL := help
GO ?= go
BINARY := bin/api

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the API locally
	$(GO) run ./cmd/api

.PHONY: build
build: ## Build the API binary
	$(GO) build -o $(BINARY) ./cmd/api

.PHONY: tidy
tidy: ## Sync go.mod / go.sum
	$(GO) mod tidy

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: test
test: ## Run all tests, including testcontainers integration (needs Docker)
	$(GO) test ./... -race -count=1

.PHONY: test-unit
test-unit: ## Run unit tests only (short mode, no Docker required)
	$(GO) test ./... -short -race -count=1

.PHONY: up
up: ## Start local Postgres + Kafka via docker compose
	docker compose up -d

.PHONY: down
down: ## Stop local Postgres + Kafka (and drop volumes)
	docker compose down -v

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin

# Targets added as milestones land: `lint` (golangci-lint, M5),
# `docker-build` (Dockerfile, M5).
