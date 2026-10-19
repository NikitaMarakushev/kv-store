# build
`go build -o kv-server cmd/server/main.go`

# run
`./kv-server`

# Check 
`ss -tlnp | grep 6381`
`curl -s http://127.0.0.1:6381/debug/pprof/ | head -n 10`

# Run pprof
`go tool pprof http://127.0.0.1:6381/debug/pprof/heap`

# Set a value
`echo -e "SET mykey hello_world\r\n" | nc localhost 6379`

# Get a value
`echo -e "GET mykey\r\n" | nc localhost 6379`

# Set with TTL 5 seconds
`echo -e "SET tempkey expires_soon 5\r\n" | nc localhost 6379`

# Run the concurrent load tester
`make load`

# Run benchmarks
`make bench`

# CPU Profiling
`make pprof-cpu`

# Heap Profiling
`make pprof-heap`

# Test
`make test`

# Clean artifacts
`make clean`

# Run demo
`make demo`