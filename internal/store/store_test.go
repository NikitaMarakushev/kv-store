package store

import (
	"fmt"
	"sync"
	"testing"
)

type NaiveStore struct {
	mu    sync.RWMutex
	items map[string]*Item
}

func NewNaiveStore() *NaiveStore {
	return &NaiveStore{items: make(map[string]*Item)}
}

func (n *NaiveStore) Set(key, value string, ttlSeconds int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.items[key] = &Item{Value: value}
}

func (n *NaiveStore) Get(key string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	item, exists := n.items[key]
	if !exists || item.IsExpired() {
		return "", false
	}
	return item.Value, true
}

func benchmarkSet(b *testing.B, store interface{ Set(string, string, int) }, numShards int) {
	keys := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		keys[i] = fmt.Sprintf("key_%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			store.Set(keys[i%10000], "value", 0)
			i++
		}
	})
}

func BenchmarkNaiveStore_Set(b *testing.B) {
	benchmarkSet(b, NewNaiveStore(), 0)
}

func BenchmarkShardedStore_Set_64(b *testing.B) {
	benchmarkSet(b, NewShardedStore(64), 64)
}

func BenchmarkShardedStore_Set_256(b *testing.B) {
	benchmarkSet(b, NewShardedStore(256), 256)
}
