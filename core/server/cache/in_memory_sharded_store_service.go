package cache

import (
	"errors"
	"sync"
	"time"
)

type shard struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	entry any
	time  int64
}

type ShardedInMemoryStore struct {
	shards         []*shard
	shardsCount    uint32
	expirationTime int64
	ticker         *time.Ticker
	done           chan any
	once           sync.Once
}

func (cache *ShardedInMemoryStore) getShard(key string) *shard {
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return cache.shards[h&(cache.shardsCount-1)]
}

func (cache *ShardedInMemoryStore) Store(key string, value any) {
	shard := cache.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.entries[key] = &cacheEntry{entry: value, time: time.Now().Unix()}
}

func (cache *ShardedInMemoryStore) Get(key string) (any, error) {
	shard := cache.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	result, present := shard.entries[key]
	if !present {
		return nil, errors.New("cache entry is not present")
	}
	result.time = time.Now().Unix()
	return result.entry, nil
}

func (cache *ShardedInMemoryStore) LoadOrStore(key string, create func() any) any {
	shard := cache.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	result, present := shard.entries[key]
	if !present {
		value := create()
		shard.entries[key] = &cacheEntry{entry: value, time: time.Now().Unix()}
		return value
	}
	result.time = time.Now().Unix()
	return result.entry
}

func (cache *ShardedInMemoryStore) Delete(key string) {
	shard := cache.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.entries, key)
}

func (cache *ShardedInMemoryStore) handleEviction() {
	currentTime := time.Now().Unix()
	for _, shard := range cache.shards {
		shard.mu.Lock()
		for key, value := range shard.entries {
			if currentTime-cache.expirationTime > value.time {
				delete(shard.entries, key)
			}
		}
		shard.mu.Unlock()
	}
}

func (cache *ShardedInMemoryStore) Close() error {
	cache.once.Do(func() { close(cache.done) })
	return nil
}

func NewCache(
	expirationTime int64,
	evictionInterval time.Duration,
	shardCount uint32,
) *ShardedInMemoryStore {
	ticker := time.NewTicker(evictionInterval)
	cache := &ShardedInMemoryStore{
		shards:         make([]*shard, shardCount),
		shardsCount:    shardCount,
		expirationTime: expirationTime,
		done:           make(chan any),
		ticker:         ticker,
	}

	for i := range shardCount {
		cache.shards[i] = &shard{entries: make(map[string]*cacheEntry)}
	}
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-cache.done:
				return
			case <-ticker.C:
				cache.handleEviction()
			}
		}
	}()

	return cache
}
