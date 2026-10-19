package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/nmarakushev/kv-store/internal/store"
)

func main() {
	db := store.NewShardedStore(256)

	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalf("Failed to start TCP server on 6379: %v", err)
	}
	defer listener.Close()
	log.Println("KV Store listening on port 6379...")

	go func() {
		log.Println("Pprof server listening on port 6381...")
		if err := http.ListenAndServe(":6381", nil); err != nil {
			log.Fatalf("Pprof server failed to start: %v", err)
		}
	}()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("\n Shutting down gracefully...")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn, db)
	}
}

func handleConnection(conn net.Conn, db *store.ShardedStore) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		response := processCommand(line, db)
		_, _ = conn.Write([]byte(response + "\r\n"))
	}
}

func processCommand(cmd string, db *store.ShardedStore) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "-ERR empty command"
	}

	command := strings.ToUpper(parts[0])

	switch command {
	case "SET":
		if len(parts) < 3 {
			return "-ERR wrong number of arguments for 'set' command"
		}
		key := parts[1]
		value := parts[2]
		ttl := 0
		if len(parts) >= 4 {
			if t, err := strconv.Atoi(parts[3]); err == nil {
				ttl = t
			}
		}
		db.Set(key, value, ttl)
		return "+OK"

	case "GET":
		if len(parts) != 2 {
			return "-ERR wrong number of arguments for 'get' command"
		}
		if val, ok := db.Get(parts[1]); ok {
			return "$" + strconv.Itoa(len(val)) + "\r\n" + val
		}
		return "$-1"

	case "DEL":
		if len(parts) != 2 {
			return "-ERR wrong number of arguments for 'del' command"
		}
		if db.Delete(parts[1]) {
			return ":1"
		}
		return ":0"

	default:
		return "-ERR unknown command '" + command + "'"
	}
}
