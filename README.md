# build
go build -o kv-server cmd/server/main.go

# run
./kv-server

# Check 
ss -tlnp | grep 6381
curl -s http://127.0.0.1:6381/debug/pprof/ | head -n 10

# Run pprof
go tool pprof http://127.0.0.1:6381/debug/pprof/heap