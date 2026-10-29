# High-Performance KV-Store

A high-performance, in-memory Key-Value store written in Go, designed with infrastructure reliability and scalability in mind. Features a sharded concurrent architecture, binary Write-Ahead Log (WAL) for durability, and a modern gRPC API.

## Features

- **High Concurrency**: Sharded in-memory storage with fine-grained `sync.RWMutex` locking to eliminate contention.
- **Durability (Crash Recovery)**: Custom binary Write-Ahead Log (WAL) ensures no data loss on unexpected process termination (`kill -9`).
- **Optimized I/O**: Batched `fsync` strategy provides a tunable trade-off between absolute durability and high throughput.
- **Modern API**: Native gRPC interface with Protobuf serialization and reflection support.
- **Observability**: Built-in `net/http/pprof` endpoints for CPU and Heap profiling.
- **Graceful Shutdown**: Proper handling of `SIGINT`/`SIGTERM` to flush buffers and close resources safely.

# Quick Start
## Prerequisites
- Go 1.21+
- `grpcurl` (optional, for manual CLI testing)

# Build & Run
### Install dependencies
`make deps`

### Build binaries
`make build`

### Run the server (listens on :6379 for gRPC, :6381 for pprof)
`make run`

# Manual Testing (via gRPC)
Since the server uses gRPC, you can interact with it using `grpcurl` (requires [grpcurl](https://github.com/fullstorydev/grpcurl)):

### Set a value
`grpcurl -plaintext -d '{"key":"mykey","value":"hello_world"}' localhost:6379 kv.KV/Set`

### Get a value
`grpcurl -plaintext -d '{"key":"mykey"}' localhost:6379 kv.KV/Get`

### Set with TTL (e.g., 5 seconds)
`grpcurl -plaintext -d '{"key":"tempkey","value":"expires_soon","ttl_seconds":5}' localhost:6379 kv.KV/Set`

### Delete a value
`grpcurl -plaintext -d '{"key":"mykey"}' localhost:6379 kv.KV/Delete`

# 📊 Performance & Load Testing
### Run the concurrent gRPC load tester (500k requests)
`make load`

### Run Go benchmarks to compare Sharded vs Naive store performance
`make bench`

### Generate and analyze CPU profile (5 seconds)
`make pprof-cpu`

### Generate and analyze Heap profile
`make pprof-heap`

# Performance & Trade-offs
- Throughput: ~60,000+ req/sec (local machine, 100 workers, gRPC).
- Binary WAL: Uses a custom Little-Endian binary protocol instead of JSON, minimizing memory allocations and disk footprint.
- Batched fsync: The fsync system call is triggered every 1,000 operations rather than on every write.
- Trade-off: In the event of a sudden power loss, a maximum of ~1,000 recent operations may be lost. However, this increases throughput by an order of magnitude compared to synchronous fsync, while still protecting against standard process crashes.
- Graceful Degradation: The WAL replay mechanism safely handles partial writes or corrupted file tails from abrupt crashes without panicking.

# Testing
### Run unit and integration tests (including Crash Recovery test)
` make test`

# Cleanup
### Remove build artifacts and profiling data
`make clean`
