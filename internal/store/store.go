package store

import (
	"hash/fnv"
	"sync"
	"time"
)

type Item struct {
	Value     string
	ExpiresAt time.Time
}

func (i *Item) IsExpired() bool {
	if i.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.ExpiresAt)
}

type Shard struct {
	mu    sync.RWMutex
	items map[string]*Item
}

type ShardedStore struct {
	shards []*Shard
	mask   uint32
}

func NewShardedStore(numShards int) *ShardedStore {
	shards := make([]*Shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &Shard{
			items: make(map[string]*Item),
		}
	}

	return &ShardedStore{
		shards: shards,
		mask:   uint32(numShards - 1),
	}
}

func (s *ShardedStore) getShardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() & s.mask
}

func (s *ShardedStore) Set(key, value string, ttlSeconds int) {
	shard := s.shards[s.getShardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	item := &Item{Value: value}
	if ttlSeconds > 0 {
		item.ExpiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	shard.items[key] = item
}

func (s *ShardedStore) Get(key string) (string, bool) {
	shard := s.shards[s.getShardIndex(key)]
	shard.mu.RLock()
	item, exists := shard.items[key]
	shard.mu.RUnlock()

	if !exists {
		return "", false
	}

	if item.IsExpired() {
		return "", false
	}

	return item.Value, true
}

func (s *ShardedStore) Delete(key string) bool {
	shard := s.shards[s.getShardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	_, exists := shard.items[key]
	if exists {
		delete(shard.items, key)
	}
	return exists
}
