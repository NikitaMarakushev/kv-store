# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOBENCH=$(GOCMD) test
BINARY_SERVER=kv-server
BINARY_LOADER=loader
PKG=./...

.PHONY: all build run test bench load pprof-cpu pprof-heap clean help

## all: Build all binaries (default target)
all: build

## build: Compile server and load tester
build:
	@echo "Building server and loader..."
	$(GOBUILD) -o $(BINARY_SERVER) cmd/server/main.go
	$(GOBUILD) -o $(BINARY_LOADER) cmd/loader/main.go

## run: Build and run the KV server
run: build
	@echo "Starting KV Server on :6379 and pprof on :6381..."
	./$(BINARY_SERVER)

## test: Run unit tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover $(PKG)

## bench: Run benchmarks to prove sharding performance
bench:
	@echo "Running benchmarks (this may take a minute)..."
	$(GOBENCH) -bench=. -benchmem ./internal/store/

## load: Run the concurrent load tester (requires server to be running)
load: build
	@echo "Starting load test..."
	./$(BINARY_LOADER)

## pprof-cpu: Generate and analyze 5-second CPU profile
pprof-cpu:
	@echo "Generating CPU profile for 5 seconds..."
	@echo "Make sure the server is running in another terminal!"
	curl -s -o cpu.prof "http://127.0.0.1:6381/debug/pprof/profile?seconds=5"
	go tool pprof -top cpu.prof

## pprof-heap: Generate and analyze Heap profile
pprof-heap:
	@echo "Generating Heap profile..."
	curl -s -o heap.prof "http://127.0.0.1:6381/debug/pprof/heap"
	go tool pprof -top heap.prof

## clean: Remove build artifacts and profiles
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_SERVER) $(BINARY_LOADER) *.prof *.pb.gz
	rm -rf pprof/

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/^## //' | column -t -s ':'

## To run the demo
demo: build
	@echo "Starting server in background..."
	@./$(BINARY_SERVER) & echo $$! > server.pid
	@sleep 2
	@echo "Running quick load test..."
	@./$(BINARY_LOADER)
	@echo "Stopping server..."
	@kill $$(cat server.pid) && rm server.pid

.DEFAULT_GOAL := help