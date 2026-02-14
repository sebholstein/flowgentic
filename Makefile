.PHONY: build build-worker build-control-plane build-hookctl build-agentctl build-acpchat lint package proto proto-lint proto-clean sqlc clean run-worker run-worker-2 run-cp run-cp-2

build: lint build-worker build-control-plane build-hookctl build-agentctl build-acpchat ## Build all binaries

lint: ## Check that all *Deps structs are fully initialized
	go tool exhaustruct -i '.*Deps$$' ./...

build-worker: ## Build the worker binary
	go build -o bin/flowgentic-worker ./cmd/flowgentic-worker

build-control-plane: ## Build the control plane binary
	go build -o bin/flowgentic-control-plane ./cmd/flowgentic-control-plane

build-hookctl: ## Build the hook binary for agent shell hooks
	go build -o bin/hookctl ./cmd/hookctl

build-agentctl: ## Build the agent-facing CLI tool
	go build -o bin/agentctl ./cmd/agentctl

build-acpchat: ## Build the ACP chat CLI
	go build -o bin/acpchat ./cmd/acpchat

package: build ## Build binaries and package the Electron app
	cd frontend && pnpm electron:package

sqlc: ## Generate Go code from SQL queries
	sqlc generate

proto: proto-lint proto-clean ## Generate Go code from proto files
	buf generate

proto-lint: ## Lint proto files
	buf lint

proto-clean: ## Remove all generated proto code
	rm -rf internal/proto/gen

run-worker: build-agentctl ## Run the worker (requires FLOWGENTIC_WORKER_SECRET)
	PATH="$(CURDIR)/bin:$(PATH)" FLOWGENTIC_WORKER_SECRET=dev-secret go run ./cmd/flowgentic-worker

run-worker-2: ## Run a second worker on port 8082
	FLOWGENTIC_WORKER_SECRET=dev-secret-2 go run ./cmd/flowgentic-worker --listen-addr=:8082

run-cp: build-worker build-agentctl ## Run the control plane (auto-starts embedded worker)
	PATH="$(CURDIR)/bin:$(PATH)" go run ./cmd/flowgentic-control-plane

run-cp-2: build-worker build-agentctl ## Run a second control plane on port 8090
	PATH="$(CURDIR)/bin:$(PATH)" go run ./cmd/flowgentic-control-plane --listen-addr=:8090

clean: proto-clean ## Remove all build artifacts and generated code
	rm -rf bin
