.PHONY: help build run test clean docker-build docker-run dev deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build the auth service
	go build -o bin/auth ./cmd/auth

run: ## Run the auth service (requires .env)
	go run ./cmd/auth/main.go

dev: ## Run with hot reload (requires air: go install github.com/air-verse/air@latest)
	air

test: ## Run tests
	go test ./... -v

test-coverage: ## Run tests with coverage
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/ tmp/ coverage.out

docker-build: ## Build docker image
	docker build -t trevecca-pedia-auth:latest .

docker-run: ## Run docker container (requires DATABASE_URL and JWT_SECRET env vars)
	docker run -p 8083:8083 \
		-e DATABASE_URL="${DATABASE_URL}" \
		-e JWT_SECRET="${JWT_SECRET}" \
		-e DEV_SEED=true \
		trevecca-pedia-auth:latest

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

fmt: ## Format code
	go fmt ./...

.DEFAULT_GOAL := help
