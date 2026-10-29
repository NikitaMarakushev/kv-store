package main

import (
	"context"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	kv "github.com/nmarakushev/kv-store/api"
	"github.com/nmarakushev/kv-store/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type KVServer struct {
	kv.UnimplementedKVServer
	db *store.ShardedStore
}

func (s *KVServer) Set(ctx context.Context, req *kv.SetRequest) (*kv.SetResponse, error) {
	s.db.Set(req.Key, req.Value, int(req.TtlSeconds))
	return &kv.SetResponse{Ok: true}, nil
}

func (s *KVServer) Get(ctx context.Context, req *kv.GetRequest) (*kv.GetResponse, error) {
	val, found := s.db.Get(req.Key)
	return &kv.GetResponse{Value: val, Found: found}, nil
}

func (s *KVServer) Delete(ctx context.Context, req *kv.DeleteRequest) (*kv.DeleteResponse, error) {
	deleted := s.db.Delete(req.Key)
	return &kv.DeleteResponse{Deleted: deleted}, nil
}

func main() {
	const walPath = "kvstore.wal"

	wal, err := store.NewWAL(walPath)
	if err != nil {
		log.Fatalf("Failed to initialize WAL: %v", err)
	}
	db := store.NewShardedStore(256, wal)

	log.Println("Replaying WAL to restore state...")
	if err := db.ReplayWAL(); err != nil {
		log.Fatalf("Failed to replay WAL: %v", err)
	}
	log.Println("WAL replay completed.")

	grpcServer := grpc.NewServer()
	kvService := &KVServer{db: db}
	kv.RegisterKVServer(grpcServer, kvService)

	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("\nShutting down gRPC server gracefully...")
		grpcServer.GracefulStop()
	}()

	go func() {
		log.Println("Pprof server listening on :6381...")
		if err := http.ListenAndServe(":6381", nil); err != nil {
			log.Fatalf("Pprof server failed: %v", err)
		}
	}()

	log.Println("KV gRPC server listening on :6379...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}

	db.Close()
}
