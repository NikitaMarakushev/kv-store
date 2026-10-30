package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	kv "github.com/nmarakushev/kv-store/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	workers := 100
	reqsPerWorker := 5000
	totalReqs := workers * reqsPerWorker

	fmt.Printf("Starting gRPC load test: %d workers, %d reqs each (%d total)\n", workers, reqsPerWorker, totalReqs)

	conn, err := grpc.NewClient("localhost:6379",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	start := time.Now()

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			client := kv.NewKVClient(conn)
			ctx := context.Background()

			for i := 0; i < reqsPerWorker; i++ {
				req := &kv.SetRequest{
					Key:   fmt.Sprintf("key_%d_%d", workerID, i),
					Value: fmt.Sprintf("value_%d", i),
				}
				_, err := client.Set(ctx, req)
				if err != nil {
					log.Printf("Set failed: %v", err)
					return
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("Load test finished in %v (%.2f req/sec)\n", duration, float64(totalReqs)/duration.Seconds())
}
