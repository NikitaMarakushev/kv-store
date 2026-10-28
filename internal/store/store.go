package store

import (
	"fmt"
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
	wal    *WAL
}

func NewShardedStore(numShards int, wal *WAL) *ShardedStore {
	shards := make([]*Shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &Shard{
			items: make(map[string]*Item),
		}
	}

	return &ShardedStore{
		shards: shards,
		mask:   uint32(numShards - 1),
		wal:    wal,
	}
}

func (s *ShardedStore) setInternal(key, value string, expiresAt time.Time) {
	shard := s.shards[s.getShardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.items[key] = &Item{Value: value, ExpiresAt: expiresAt}
}

func (s *ShardedStore) deleteInternal(key string) bool {
	shard := s.shards[s.getShardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	_, exists := shard.items[key]
	if exists {
		delete(shard.items, key)
	}
	return exists
}

func (s *ShardedStore) ReplayWAL() error {
	if s.wal == nil {
		return nil
	}
	return s.wal.Replay(func(entry WALEntry) error {
		switch entry.Op {
		case "SET":
			var expiresAt time.Time
			if !entry.ExpiresAt.IsZero() {
				if time.Now().After(entry.ExpiresAt) {
					return nil
				}
				expiresAt = entry.ExpiresAt
			}

			s.setInternal(entry.Key, entry.Value, expiresAt)

		case "DEL":
			s.deleteInternal(entry.Key)
		}
		return nil
	})
}

func (s *ShardedStore) getShardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() & s.mask
}

func (s *ShardedStore) Set(key, value string, ttlSeconds int) {
	var expiresAt time.Time
	if ttlSeconds > 0 {
		expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	if s.wal != nil {
		err := s.wal.Append(WALEntry{
			Op:        "SET",
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			panic(fmt.Sprintf("WAL append failed: %v", err))
		}
	}

	s.setInternal(key, value, expiresAt)
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
	if s.wal != nil {
		_ = s.wal.Append(WALEntry{Op: "DEL", Key: key})
	}

	return s.deleteInternal(key)
}

func (s *ShardedStore) Close() {
	if s.wal != nil {
		_ = s.wal.Close()
	}
}
